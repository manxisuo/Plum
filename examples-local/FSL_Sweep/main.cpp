#include <csignal>
#include <cstdlib>
#include <iostream>
#include <string>

#include <QCoreApplication>
#include <QDateTime>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonParseError>
#include <QRandomGenerator>
#include <QThread>

#include "plumworker/stream_worker.hpp"
#include "../FSL_Common/simulation_utils.hpp"
#include "../FSL_Common/httplib.h"

using namespace plumworker;

namespace {

StreamWorker *g_worker = nullptr;

class StageProgressSender {
public:
    StageProgressSender(const QString &taskId, const QString &stage)
        : taskId_(taskId), stage_(stage) {
        const char *env = std::getenv("MAIN_CONTROL_BASE");
        base_ = env ? env : "http://127.0.0.1:4000";
        if (!base_.empty() && base_.back() == '/') {
            base_.pop_back();
        }
    }

    bool send(const QVector<TingState> &tings,
              const QJsonArray &trackChunk = QJsonArray(),
              const QJsonArray *suspects = nullptr,
              const QJsonArray *confirmed = nullptr,
              const QJsonArray *cleared = nullptr,
              const QJsonArray *destroyed = nullptr,
              const QJsonArray *evaluated = nullptr) {
        if (trackChunk.isEmpty() && !suspects && !confirmed && !cleared && !destroyed && !evaluated) {
            return false;
        }

        QJsonObject body;
        body.insert("tings", serializeTings(tings));
        if (!trackChunk.isEmpty()) {
            body.insert("tracks", trackChunk);
        }
        if (suspects) {
            body.insert("suspect_mines", *suspects);
        }
        if (confirmed) {
            body.insert("confirmed_mines", *confirmed);
        }
        if (cleared) {
            body.insert("cleared_mines", *cleared);
        }
        if (destroyed) {
            body.insert("destroyed_mines", *destroyed);
        }
        if (evaluated) {
            body.insert("evaluated_mines", *evaluated);
        }

        QJsonDocument doc(body);
        const std::string path =
            "/api/task/" + taskId_.toStdString() + "/stage/" + stage_.toStdString() + "/progress";

        try {
            httplib::Client client(base_.c_str());
            client.set_connection_timeout(1, 0);
            client.set_read_timeout(2, 0);
            auto res = client.Post(path.c_str(),
                                   doc.toJson(QJsonDocument::Compact).toStdString(),
                                   "application/json");
            if (!res || res->status >= 300) {
                std::cerr << "[FSL_Sweep] 进度上报失败 status="
                          << (res ? res->status : 0) << std::endl;
                return false;
            }
            sent_ = true;
            return true;
        } catch (const std::exception &ex) {
            std::cerr << "[FSL_Sweep] 进度上报异常: " << ex.what() << std::endl;
            return false;
        }
    }

    bool sent() const { return sent_; }

private:
    QString taskId_;
    QString stage_;
    std::string base_;
    bool sent_{false};
};

QString handleSweepTask(const QString &controllerTaskId, const QString &payload) {
    QJsonParseError error;
    const QJsonDocument doc = QJsonDocument::fromJson(payload.toUtf8(), &error);
    if (error.error != QJsonParseError::NoError || !doc.isObject()) {
        throw std::runtime_error(QString("JSON 解析失败: %1").arg(error.errorString()).toStdString());
    }

    const QJsonObject root = doc.object();
    const QString mainTaskId = root.value("task_id").toString();
    const QJsonArray tingsArray = root.value("tings").toArray();
    const QJsonArray zonesArray = root.value("work_zones").toArray();

    if (tingsArray.isEmpty() || zonesArray.isEmpty()) {
        throw std::runtime_error("tings 或 work_zones 不能为空");
    }

    QVector<TingState> tings = parseTings(tingsArray);
    QVector<WorkZone> zones = parseZones(zonesArray);
    if (zones.size() < tings.size()) {
        throw std::runtime_error("作业区数量不足，必须与艇数量一致");
    }

    const int seed = root.value("random_seed").toInt(QDateTime::currentMSecsSinceEpoch() & 0xFFFFFFFF);
    QRandomGenerator rng(static_cast<quint32>(seed));
    const QDateTime phaseStart = QDateTime::currentDateTimeUtc();

    constexpr int kStepDelayMs = 150;
    constexpr int kProgressBatchSize = 4;

    QVector<QJsonArray> tingTracks(tings.size());
    QJsonArray revealedSuspects;
    QJsonArray revealedConfirmed;
    bool hasGlobalSuspect = false;

    struct PendingMine {
        MineInfo info;
        bool revealed = false;
    };
    QVector<QVector<PendingMine>> pendingMines(tings.size());

    StageProgressSender progressSender(mainTaskId.isEmpty() ? controllerTaskId : mainTaskId, "sweep");

    for (int i = 0; i < tings.size(); ++i) {
        TingState &ting = tings[i];
        const WorkZone &zone = zones[i % zones.size()];

        const double centerLon = (zone.topLeft.lon + zone.bottomRight.lon) / 2.0;
        GeoPoint zoneEntry{zone.topLeft.lat, centerLon};
        GeoPoint zoneExit{zone.bottomRight.lat, centerLon};

        QJsonArray &track = tingTracks[i];
        appendLinearTrack(track, ting.id, "sweep", ting.position, zoneEntry, ting.speedMps,
                          ting.elapsedSeconds, phaseStart);
        ting.position = zoneEntry;

        appendLinearTrack(track, ting.id, "sweep", zoneEntry, zoneExit, ting.speedMps,
                          ting.elapsedSeconds, phaseStart);
        ting.position = zoneExit;

        const int targetCount = rng.bounded(2, 5); // 2 ~ 4 个目标
        QList<MineInfo> zoneSuspects;
        QList<MineInfo> zoneConfirmed;

        int attempts = 0;
        while ((zoneSuspects.size() + zoneConfirmed.size()) < targetCount && attempts < targetCount * 10) {
            ++attempts;
            const double latRatio = rng.generateDouble() * 0.7 + 0.15;
            const double lonRatio = rng.generateDouble() * 0.5 + 0.25;

            GeoPoint minePos;
            minePos.lat = zone.bottomRight.lat + latRatio * (zone.topLeft.lat - zone.bottomRight.lat);
            minePos.lon = zone.topLeft.lon + lonRatio * (zone.bottomRight.lon - zone.topLeft.lon);

            const GeoPoint projection{minePos.lat, centerLon};
            const double lateralDistance = haversineDistanceMeters(minePos, projection);
            if (lateralDistance > ting.sonarRange * 0.9) {
                continue;
            }

            MineInfo mine;
            mine.id = QString("mine_%1_%2").arg(ting.id).arg(zoneSuspects.size() + zoneConfirmed.size() + 1);
            mine.position = minePos;
            mine.assignedTing = ting.id;

            const double roll = rng.generateDouble();
            const bool confirmed = roll < ting.confirmProb;
            if (confirmed) {
                mine.status = "confirmed";
                zoneConfirmed.append(mine);
            } else {
                mine.status = "suspect";
                zoneSuspects.append(mine);
            }
        }

        if (zoneSuspects.isEmpty() && !zoneConfirmed.isEmpty()) {
            MineInfo mine = zoneConfirmed.takeFirst();
            mine.status = "suspect";
            zoneSuspects.append(mine);
        }

        for (const auto &mine : zoneSuspects) {
            PendingMine pending;
            pending.info = mine;
            pendingMines[i].append(pending);
        }
        for (const auto &mine : zoneConfirmed) {
            PendingMine pending;
            pending.info = mine;
            pendingMines[i].append(pending);
        }
        if (!zoneSuspects.isEmpty()) {
            hasGlobalSuspect = true;
        }
    }

    if (!hasGlobalSuspect) {
        for (auto &list : pendingMines) {
            for (auto &pending : list) {
                if (pending.info.status == "confirmed") {
                    pending.info.status = "suspect";
                    hasGlobalSuspect = true;
                    break;
                }
            }
            if (hasGlobalSuspect) {
                break;
            }
        }
    }

    auto revealMine = [&](PendingMine &pending) -> bool {
        if (pending.revealed) {
            return false;
        }
        pending.revealed = true;
        const QJsonObject mineObj = mineToJson(pending.info);
        if (pending.info.status == "confirmed") {
            revealedConfirmed.append(mineObj);
        } else {
            revealedSuspects.append(mineObj);
        }
        return true;
    };

    QVector<int> indices(tings.size(), 0);
    while (true) {
        bool advanced = false;
        QJsonArray chunk;
        bool revealChanged = false;

        for (int i = 0; i < tings.size(); ++i) {
            QJsonArray &track = tingTracks[i];
            if (indices[i] >= track.size()) {
                continue;
            }

            const QJsonObject point = track.at(indices[i]).toObject();
            indices[i]++;
            chunk.append(point);

            const QJsonObject posObj = point.value("position").toObject();
            TingState &ting = tings[i];
            ting.position.lat = posObj.value("lat").toDouble();
            ting.position.lon = posObj.value("lon").toDouble();

            for (PendingMine &pending : pendingMines[i]) {
                if (pending.revealed) {
                    continue;
                }
                const double distance = haversineDistanceMeters(ting.position, pending.info.position);
                if (distance <= ting.sonarRange) {
                    if (revealMine(pending)) {
                        revealChanged = true;
                    }
                }
            }

            if (chunk.size() >= kProgressBatchSize) {
                progressSender.send(tings, chunk, &revealedSuspects, &revealedConfirmed, nullptr, nullptr, nullptr);
                chunk = QJsonArray();
                revealChanged = false;
            }
            advanced = true;
        }

        if (!chunk.isEmpty() || revealChanged) {
            progressSender.send(tings, chunk, &revealedSuspects, &revealedConfirmed, nullptr, nullptr, nullptr);
        }

        if (!advanced) {
            break;
        }
        QThread::msleep(kStepDelayMs);
    }

    for (auto &list : pendingMines) {
        for (PendingMine &pending : list) {
            if (!pending.revealed) {
                revealMine(pending);
            }
        }
    }

    QJsonObject result;
    result.insert("status", "success");
    result.insert("tings", serializeTings(tings));
    result.insert("suspect_mines", revealedSuspects);
    result.insert("confirmed_mines", revealedConfirmed);
    result.insert("cleared_mines", QJsonArray());
    result.insert("destroyed_mines", QJsonArray());
    result.insert("evaluated_mines", QJsonArray());
    if (!progressSender.sent()) {
        QJsonArray combined;
        for (const QJsonArray &track : tingTracks) {
            for (const QJsonValue &value : track) {
                combined.append(value);
            }
        }
        result.insert("tracks", combined);
    } else {
        result.insert("tracks", QJsonArray());
    }

    return QString::fromUtf8(QJsonDocument(result).toJson(QJsonDocument::Compact));
}

void signalHandler(int sig) {
    std::cout << "[FSL_Sweep] 捕获信号 " << sig << "，准备退出..." << std::endl;
    if (g_worker) {
        g_worker->stop();
    }
}

} // namespace

int main(int argc, char *argv[]) {
    QCoreApplication app(argc, argv);

    std::signal(SIGINT, signalHandler);
    std::signal(SIGTERM, signalHandler);

    StreamWorkerOptions options;
    options.labels["phase"] = "sweep";

    StreamWorker worker(options);
    g_worker = &worker;

    worker.registerTask(
        "扫雷",
        [](const std::string &taskId, const std::string &taskName, const std::string &payload) -> std::string {
            Q_UNUSED(taskName);
            try {
                const QString response = handleSweepTask(QString::fromStdString(taskId),
                                                         QString::fromUtf8(payload.c_str()));
                return response.toStdString();
            } catch (const std::exception &ex) {
                QJsonObject error;
                error.insert("status", "error");
                error.insert("message", QString::fromUtf8(ex.what()));
                return QJsonDocument(error).toJson(QJsonDocument::Compact).toStdString();
            }
        });

    worker.start();
    std::cout << "[FSL_Sweep] 已退出" << std::endl;
    return 0;
}

