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
                std::cerr << "[FSL_Destroy] 进度上报失败 status="
                          << (res ? res->status : 0) << std::endl;
                return false;
            }
            sent_ = true;
            return true;
        } catch (const std::exception &ex) {
            std::cerr << "[FSL_Destroy] 进度上报异常: " << ex.what() << std::endl;
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

QVector<MineInfo> parseConfirmedMines(const QJsonArray &array) {
    QVector<MineInfo> mines;
    mines.reserve(array.size());
    for (const auto &item : array) {
        const QJsonObject obj = item.toObject();
        MineInfo mine;
        mine.id = obj.value("id").toString();
        mine.position = geoPointFromJson(obj.value("position").toObject(), "confirmed.position");
        mine.status = obj.value("status").toString("confirmed");
        mine.assignedTing = obj.value("assigned_ting").toString();
        mines.append(mine);
    }
    return mines;
}

QString handleDestroyTask(const QString &controllerTaskId, const QString &payload) {
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
    QVector<MineInfo> confirmedMines = parseConfirmedMines(root.value("confirmed_mines").toArray());

    if (confirmedMines.isEmpty()) {
        QJsonObject result;
        result.insert("status", "success");
        result.insert("tings", serializeTings(tings));
        result.insert("destroyed_mines", QJsonArray());
        result.insert("tracks", QJsonArray());
        return QString::fromUtf8(QJsonDocument(result).toJson(QJsonDocument::Compact));
    }

    const int seed = root.value("random_seed").toInt(QDateTime::currentMSecsSinceEpoch() & 0xFFFFFFFF);
    QRandomGenerator rng(static_cast<quint32>(seed));
    const QDateTime phaseStart = QDateTime::currentDateTimeUtc();

    constexpr int kStepDelayMs = 150;
    constexpr int kProgressBatchSize = 4;

    QVector<QJsonArray> tingTracks(tings.size());
    QJsonArray destroyedArray;
    QVector<bool> processed(confirmedMines.size(), false);

    StageProgressSender progressSender(mainTaskId.isEmpty() ? controllerTaskId : mainTaskId, "destroy");

    struct PendingDestroy {
        MineInfo info;
        bool revealed = false;
    };
    QVector<QVector<PendingDestroy>> pendingDestroys(tings.size());

    auto revealDestroy = [&](int tingIdx, const TingState &state, bool force = false) -> bool {
        constexpr double kRevealRange = 12.0;
        bool changed = false;
        for (PendingDestroy &pending : pendingDestroys[tingIdx]) {
            if (pending.revealed) {
                continue;
            }
            if (!force) {
                const double distance = haversineDistanceMeters(state.position, pending.info.position);
                if (distance > kRevealRange) {
                    continue;
                }
            }
            pending.revealed = true;
            destroyedArray.append(mineToJson(pending.info));
            changed = true;
        }
        return changed;
    };

    while (true) {
        bool assigned = false;

        for (int i = 0; i < tings.size(); ++i) {
            TingState &ting = tings[i];

            int targetIndex = -1;
            double shortest = std::numeric_limits<double>::max();
            for (int m = 0; m < confirmedMines.size(); ++m) {
                if (processed[m]) {
                    continue;
                }
                const double dist = haversineDistanceMeters(ting.position, confirmedMines[m].position);
                if (dist < shortest) {
                    shortest = dist;
                    targetIndex = m;
                }
            }

            if (targetIndex < 0) {
                continue;
            }

            assigned = true;
            MineInfo &mine = confirmedMines[targetIndex];
            QJsonArray &track = tingTracks[i];

            const double travel = estimateTravelTimeSeconds(ting.position, mine.position, ting.speedMps);
            appendLinearTrack(track, ting.id, "destroy", ting.position, mine.position,
                              ting.speedMps, ting.elapsedSeconds, phaseStart);
            ting.position = mine.position;

            const double dwellSeconds = 5.0 + rng.generateDouble() * 4.0;
            appendDwellTrack(track, ting.id, "destroy", ting.position, ting.elapsedSeconds,
                             phaseStart, dwellSeconds, 8);

            mine.status = "destroyed";
            mine.assignedTing = ting.id;

            PendingDestroy pending;
            pending.info = mine;
            pendingDestroys[i].append(pending);

            processed[targetIndex] = true;
        }

        if (!assigned) {
            break;
        }
    }

    QVector<int> indices(tings.size(), 0);
    while (true) {
        bool advanced = false;
        QJsonArray chunk;
        bool anyReveal = false;

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

            if (revealDestroy(i, ting)) {
                anyReveal = true;
            }

            if (chunk.size() >= kProgressBatchSize) {
                progressSender.send(tings, chunk, nullptr, nullptr, nullptr, &destroyedArray, nullptr);
                chunk = QJsonArray();
                anyReveal = false;
            }
            advanced = true;
        }

        if (!chunk.isEmpty() || anyReveal) {
            progressSender.send(tings, chunk, nullptr, nullptr, nullptr, &destroyedArray, nullptr);
        }

        if (!advanced) {
            break;
        }
        QThread::msleep(kStepDelayMs);
    }

    for (int i = 0; i < tings.size(); ++i) {
        revealDestroy(i, tings[i], true);
    }

    QJsonObject result;
    result.insert("status", "success");
    result.insert("tings", serializeTings(tings));
    result.insert("suspect_mines", QJsonArray());
    result.insert("confirmed_mines", QJsonArray());
    result.insert("cleared_mines", QJsonArray());
    result.insert("destroyed_mines", destroyedArray);
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
    std::cout << "[FSL_Destroy] 捕获信号 " << sig << "，准备退出..." << std::endl;
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
    options.labels["phase"] = "destroy";

    StreamWorker worker(options);
    g_worker = &worker;

    worker.registerTask(
        "灭雷",
        [](const std::string &taskId, const std::string &taskName, const std::string &payload) -> std::string {
            Q_UNUSED(taskName);
            try {
                const QString response = handleDestroyTask(QString::fromStdString(taskId),
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
    std::cout << "[FSL_Destroy] 已退出" << std::endl;
    return 0;
}

