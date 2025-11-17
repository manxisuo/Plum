#include <csignal>
#include <cstdlib>
#include <iostream>
#include <limits>
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
                std::cerr << "[FSL_Investigate] 进度上报失败 status="
                          << (res ? res->status : 0) << std::endl;
                return false;
            }
            sent_ = true;
            return true;
        } catch (const std::exception &ex) {
            std::cerr << "[FSL_Investigate] 进度上报异常: " << ex.what() << std::endl;
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

struct PendingMine {
    MineInfo info;
    bool processed = false;
};

QVector<PendingMine> parseSuspects(const QJsonArray &array) {
    QVector<PendingMine> mines;
    mines.reserve(array.size());
    for (const auto &item : array) {
        const QJsonObject obj = item.toObject();
        PendingMine mine;
        mine.info.id = obj.value("id").toString();
        mine.info.position = geoPointFromJson(obj.value("position").toObject(), "suspect.position");
        mine.info.status = obj.value("status").toString("suspect");
        mine.info.assignedTing = obj.value("assigned_ting").toString();
        mines.append(mine);
    }
    return mines;
}

QJsonArray cloneArray(const QJsonArray &array) {
    QJsonArray copy;
    for (const auto &item : array) {
        copy.append(item);
    }
    return copy;
}

QString handleInvestigateTask(const QString &controllerTaskId, const QString &payload) {
    QJsonParseError error;
    const QJsonDocument doc = QJsonDocument::fromJson(payload.toUtf8(), &error);
    if (error.error != QJsonParseError::NoError || !doc.isObject()) {
        throw std::runtime_error(QString("JSON 解析失败: %1").arg(error.errorString()).toStdString());
    }

    const QJsonObject root = doc.object();
    const QString mainTaskId = root.value("task_id").toString();
    const QJsonArray tingsArray = root.value("tings").toArray();
    if (tingsArray.isEmpty()) {
        throw std::runtime_error("tings 不能为空");
    }

    QVector<TingState> tings = parseTings(tingsArray);
    QVector<PendingMine> suspects = parseSuspects(root.value("suspect_mines").toArray());
    QJsonArray confirmedCarry = cloneArray(root.value("confirmed_mines").toArray());

    if (suspects.isEmpty()) {
        QJsonObject result;
        result.insert("status", "success");
        result.insert("tings", serializeTings(tings));
        result.insert("confirmed_mines", confirmedCarry);
        result.insert("cleared_mines", QJsonArray());
        result.insert("tracks", QJsonArray());
        return QString::fromUtf8(QJsonDocument(result).toJson(QJsonDocument::Compact));
    }

    const int seed = root.value("random_seed").toInt(QDateTime::currentMSecsSinceEpoch() & 0xFFFFFFFF);
    QRandomGenerator rng(static_cast<quint32>(seed));
    const QDateTime phaseStart = QDateTime::currentDateTimeUtc();

    constexpr int kStepDelayMs = 150;
    constexpr int kProgressBatchSize = 4;

    QVector<QJsonArray> tingTracks(tings.size());
    QJsonArray revealedConfirmed = confirmedCarry;
    QJsonArray revealedCleared;
    QJsonArray remainingSuspects = cloneArray(root.value("suspect_mines").toArray());

    StageProgressSender progressSender(mainTaskId.isEmpty() ? controllerTaskId : mainTaskId, "investigate");

    struct PendingInvestigation {
        MineInfo info;
        QString finalStatus;
        bool revealed = false;
    };
    QVector<QVector<PendingInvestigation>> pendingInvestigations(tings.size());

    auto revealInvestigation = [&](int tingIdx, const TingState &tingState, bool force = false) -> bool {
        bool changed = false;
        const double revealRange = 15.0;

        for (PendingInvestigation &pending : pendingInvestigations[tingIdx]) {
            if (pending.revealed) {
                continue;
            }

            const double distance = haversineDistanceMeters(tingState.position, pending.info.position);
            if (!force && distance > revealRange) {
                continue;
            }

            pending.revealed = true;
            for (int idx = 0; idx < remainingSuspects.size(); ++idx) {
                const QJsonObject obj = remainingSuspects.at(idx).toObject();
                if (obj.value("id").toString() == pending.info.id) {
                    remainingSuspects.removeAt(idx);
                    break;
                }
            }

            pending.info.status = pending.finalStatus;
            const QJsonObject obj = mineToJson(pending.info);
            if (pending.finalStatus == "confirmed") {
                revealedConfirmed.append(obj);
            } else if (pending.finalStatus == "cleared") {
                revealedCleared.append(obj);
            } else {
                remainingSuspects.append(obj);
            }
            changed = true;
        }

        return changed;
    };

    while (true) {
        bool assigned = false;

        for (int i = 0; i < tings.size(); ++i) {
            TingState &ting = tings[i];

            PendingMine *target = nullptr;
            double shortest = std::numeric_limits<double>::max();
            for (auto &mine : suspects) {
                if (mine.processed) {
                    continue;
                }
                const double dist = haversineDistanceMeters(ting.position, mine.info.position);
                if (dist < shortest) {
                    shortest = dist;
                    target = &mine;
                }
            }

            if (!target) {
                continue;
            }

            assigned = true;
        const double travel = estimateTravelTimeSeconds(ting.position, target->info.position, ting.speedMps);
        QJsonArray &track = tingTracks[i];
        appendLinearTrack(track, ting.id, "investigate", ting.position, target->info.position,
                          ting.speedMps, ting.elapsedSeconds, phaseStart);
            ting.position = target->info.position;

            const double dwellSeconds = 4.0 + rng.generateDouble() * 3.0;
            appendDwellTrack(track, ting.id, "investigate", ting.position, ting.elapsedSeconds,
                             phaseStart, dwellSeconds, 8);

            const double roll = rng.generateDouble();
            if (roll < ting.confirmProb) {
                target->info.status = "confirmed";
                target->info.assignedTing = ting.id;
            } else {
                target->info.status = "cleared";
                target->info.assignedTing = ting.id;
            }
            PendingInvestigation pending;
            pending.info = target->info;
            pending.finalStatus = target->info.status;
            pendingInvestigations[i].append(pending);
            target->processed = true;
        }

        if (!assigned) {
            break;
        }
    }

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

            if (revealInvestigation(i, ting)) {
                revealChanged = true;
            }

            if (chunk.size() >= kProgressBatchSize) {
                progressSender.send(tings, chunk, &remainingSuspects, &revealedConfirmed, &revealedCleared, nullptr, nullptr);
                chunk = QJsonArray();
                revealChanged = false;
            }
            advanced = true;
        }

        if (!chunk.isEmpty() || revealChanged) {
            progressSender.send(tings, chunk, &remainingSuspects, &revealedConfirmed, &revealedCleared, nullptr, nullptr);
        }

        if (!advanced) {
            break;
        }
        QThread::msleep(kStepDelayMs);
    }

    for (int i = 0; i < tings.size(); ++i) {
        revealInvestigation(i, tings[i], true);
    }

    QJsonObject result;
    result.insert("status", "success");
    result.insert("tings", serializeTings(tings));
    result.insert("suspect_mines", remainingSuspects);
    result.insert("confirmed_mines", revealedConfirmed);
    result.insert("cleared_mines", revealedCleared);
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
    std::cout << "[FSL_Investigate] 捕获信号 " << sig << "，准备退出..." << std::endl;
    if (g_worker) {
        g_worker->stop();
    }
}

}  // namespace

int main(int argc, char *argv[]) {
    QCoreApplication app(argc, argv);

    std::signal(SIGINT, signalHandler);
    std::signal(SIGTERM, signalHandler);

    StreamWorkerOptions options;
    options.labels["phase"] = "investigate";

    StreamWorker worker(options);
    g_worker = &worker;

    worker.registerTask(
        "查证",
        [](const std::string &taskId, const std::string &taskName, const std::string &payload) -> std::string {
            Q_UNUSED(taskName);
            try {
                const QString response = handleInvestigateTask(QString::fromStdString(taskId),
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
    std::cout << "[FSL_Investigate] 已退出" << std::endl;
    return 0;
}

