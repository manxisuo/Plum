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
            // 增加超时时间：连接超时5秒，读取超时5秒，确保有足够时间建立连接和接收响应
            client.set_connection_timeout(5, 0);
            client.set_read_timeout(5, 0);
            auto res = client.Post(path.c_str(),
                                   doc.toJson(QJsonDocument::Compact).toStdString(),
                                   "application/json");
            if (!res) {
                // 连接失败（res为nullptr），可能是网络问题或服务不可达
                // 不打印错误日志，避免日志过多，但返回false表示本次上报失败
                // 任务会继续执行，不会因为进度上报失败而中断
                return false;
            }
            if (res->status >= 300) {
                std::cerr << "[FSL_Evaluate] 进度上报失败 status=" << res->status << std::endl;
                return false;
            }
            sent_ = true;
            return true;
        } catch (const std::exception &ex) {
            // 异常情况下也不打印日志，避免日志过多
            // 进度上报失败不应该影响任务执行
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

struct PendingEvaluation {
    MineInfo mine;
    double score = 0.0;
    bool revealed = false;
};

QString handleEvaluateTask(const QString &controllerTaskId, const QString &payload) {
    QJsonParseError error;
    const QJsonDocument doc = QJsonDocument::fromJson(payload.toUtf8(), &error);
    if (error.error != QJsonParseError::NoError || !doc.isObject()) {
        throw std::runtime_error(QString("JSON 解析失败: %1").arg(error.errorString()).toStdString());
    }

    const QJsonObject root = doc.object();
    const QString mainTaskId = root.value("task_id").toString();
    const QJsonArray tingsArray = root.value("tings").toArray();
    const QJsonArray destroyedArray = root.value("destroyed_mines").toArray();

    if (tingsArray.isEmpty()) {
        throw std::runtime_error("tings 不能为空");
    }
    if (destroyedArray.isEmpty()) {
        QJsonObject result;
        result.insert("status", "success");
        result.insert("tings", tingsArray);
        result.insert("destroyed_mines", QJsonArray());
        result.insert("evaluated_mines", QJsonArray());
        result.insert("tracks", QJsonArray());
        return QString::fromUtf8(QJsonDocument(result).toJson(QJsonDocument::Compact));
    }

    QVector<TingState> tings = parseTings(tingsArray);
    QVector<MineInfo> destroyedMines = parseMines(destroyedArray, "destroyed");

    const int seed = root.value("random_seed").toInt(QDateTime::currentMSecsSinceEpoch() & 0xFFFFFFFF);
    QRandomGenerator rng(static_cast<quint32>(seed));
    const QDateTime phaseStart = QDateTime::currentDateTimeUtc();

    constexpr int kStepDelayMs = 150;
    constexpr int kProgressBatchSize = 4;

    QVector<QJsonArray> tingTracks(tings.size());
    QVector<bool> processed(destroyedMines.size(), false);
    QVector<QVector<PendingEvaluation>> pendingEvaluations(tings.size());
    QJsonArray revealedEvaluations;

    StageProgressSender progressSender(mainTaskId.isEmpty() ? controllerTaskId : mainTaskId, "evaluate");

    auto revealEvaluation = [&](int tingIdx, const TingState &state, bool force = false) -> bool {
        constexpr double kRevealRange = 12.0;
        bool changed = false;
        for (PendingEvaluation &pending : pendingEvaluations[tingIdx]) {
            if (pending.revealed) {
                continue;
            }
            if (!force) {
                const double distance = haversineDistanceMeters(state.position, pending.mine.position);
                if (distance > kRevealRange) {
                    continue;
                }
            }
            pending.revealed = true;
            QJsonObject obj = mineToJson(pending.mine);
            obj.insert("evaluation_score", pending.score);
            revealedEvaluations.append(obj);
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
            for (int m = 0; m < destroyedMines.size(); ++m) {
                if (processed[m]) {
                    continue;
                }
                const double dist = haversineDistanceMeters(ting.position, destroyedMines[m].position);
                if (dist < shortest) {
                    shortest = dist;
                    targetIndex = m;
                }
            }

            if (targetIndex < 0) {
                continue;
            }

            assigned = true;
            MineInfo &mine = destroyedMines[targetIndex];
            QJsonArray &track = tingTracks[i];

            appendLinearTrack(track, ting.id, "evaluate", ting.position, mine.position,
                              ting.speedMps, ting.elapsedSeconds, phaseStart);
            ting.position = mine.position;

            const double dwellSeconds = 3.0 + rng.generateDouble() * 3.0;
            appendDwellTrack(track, ting.id, "evaluate", ting.position, ting.elapsedSeconds,
                             phaseStart, dwellSeconds, 8);

            PendingEvaluation pending;
            pending.mine = mine;
            pending.score = std::round(70.0 + rng.generateDouble() * 30.0);
            pendingEvaluations[i].append(pending);

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

            if (revealEvaluation(i, ting)) {
                anyReveal = true;
            }

            if (chunk.size() >= kProgressBatchSize) {
                progressSender.send(tings, chunk, nullptr, nullptr, nullptr, nullptr, &revealedEvaluations);
                chunk = QJsonArray();
                anyReveal = false;
            }
            advanced = true;
        }

        if (!chunk.isEmpty() || anyReveal) {
            progressSender.send(tings, chunk, nullptr, nullptr, nullptr, nullptr, &revealedEvaluations);
        }

        if (!advanced) {
            break;
        }
        QThread::msleep(kStepDelayMs);
    }

    for (int i = 0; i < tings.size(); ++i) {
        revealEvaluation(i, tings[i], true);
    }

    QJsonArray finalDestroyed;
    for (int i = 0; i < pendingEvaluations.size(); ++i) {
        for (const PendingEvaluation &pending : pendingEvaluations[i]) {
            QJsonObject obj = mineToJson(pending.mine);
            obj.insert("evaluation_score", pending.score);
            finalDestroyed.append(obj);
        }
    }

    QJsonObject result;
    result.insert("status", "success");
    result.insert("tings", serializeTings(tings));
    result.insert("suspect_mines", QJsonArray());
    result.insert("confirmed_mines", QJsonArray());
    result.insert("cleared_mines", QJsonArray());
    result.insert("destroyed_mines", finalDestroyed);
    result.insert("evaluated_mines", revealedEvaluations);
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
    std::cout << "[FSL_Evaluate] 捕获信号 " << sig << "，准备退出..." << std::endl;
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
    options.labels["phase"] = "evaluate";

    StreamWorker worker(options);
    g_worker = &worker;

    worker.registerTask(
        "评估",
        [](const std::string &taskId, const std::string &taskName, const std::string &payload) -> std::string {
            Q_UNUSED(taskName);
            try {
                const QString response = handleEvaluateTask(QString::fromStdString(taskId),
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
    std::cout << "[FSL_Evaluate] 已退出" << std::endl;
    return 0;
}
