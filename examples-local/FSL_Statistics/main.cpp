#include <QCoreApplication>
#include <QFile>
#include <QTextStream>
#include <QDebug>
#include <QStringList>
#include <cmath>

#include "../FSL_Common/httplib.h"
#include "../FSL_Common/json.hpp"

#include <iostream>
#include <stdexcept>
#include <vector>
#include <algorithm>
#include <map>

using json = nlohmann::json;

static int loadPortFromMeta(int defaultPort)
{
    const QStringList candidates = {
        QCoreApplication::applicationDirPath() + "/meta.ini",
        QCoreApplication::applicationDirPath() + "/../meta.ini",
        QCoreApplication::applicationDirPath() + "/../../meta.ini"
    };

    for (const QString &path : candidates) {
        QFile file(path);
        if (!file.exists()) {
            continue;
        }
        if (!file.open(QIODevice::ReadOnly | QIODevice::Text)) {
            qWarning() << "[FSL_Statistics] 无法打开 meta.ini:" << path << "-" << file.errorString();
            continue;
        }
        QTextStream in(&file);
        while (!in.atEnd()) {
            const QString line = in.readLine().trimmed();
            if (line.isEmpty() || line.startsWith('#')) {
                continue;
            }
            if (!line.startsWith("service=")) {
                continue;
            }
            const QString spec = line.mid(QStringLiteral("service=").size()).trimmed();
            const QStringList parts = spec.split(':');
            if (parts.size() < 3) {
                continue;
            }
            if (parts[0].trimmed() == "analyzeTask") {
                bool ok = false;
                const int port = parts[2].trimmed().toInt(&ok);
                if (ok) {
                    return port;
                }
            }
        }
    }

    qWarning() << "[FSL_Statistics] 未在 meta.ini 中找到 analyzeTask 服务端口，使用默认端口" << defaultPort;
    return defaultPort;
}

// 计算两点间距离（Haversine公式，简化版）
static double calculateDistance(const json &p1, const json &p2)
{
    if (!p1.contains("lat") || !p1.contains("lon") || !p2.contains("lat") || !p2.contains("lon")) {
        return 0.0;
    }
    
    const double lat1 = p1["lat"].get<double>();
    const double lon1 = p1["lon"].get<double>();
    const double lat2 = p2["lat"].get<double>();
    const double lon2 = p2["lon"].get<double>();
    
    const double dLat = (lat2 - lat1) * M_PI / 180.0;
    const double dLon = (lon2 - lon1) * M_PI / 180.0;
    const double a = sin(dLat / 2) * sin(dLat / 2) +
                     cos(lat1 * M_PI / 180.0) * cos(lat2 * M_PI / 180.0) *
                     sin(dLon / 2) * sin(dLon / 2);
    const double c = 2 * atan2(sqrt(a), sqrt(1 - a));
    const double R = 6371000; // 地球半径（米）
    return R * c;
}

static json analyzeTask(const json &payload)
{
    json result;
    result["task_id"] = payload.value("task_id", "");
    result["stage"] = payload.value("stage", "");
    
    // 基础统计
    json summary;
    const json &tings = payload.value("tings", json::array());
    const json &suspectMines = payload.value("suspect_mines", json::array());
    const json &confirmedMines = payload.value("confirmed_mines", json::array());
    const json &clearedMines = payload.value("cleared_mines", json::array());
    const json &destroyedMines = payload.value("destroyed_mines", json::array());
    const json &evaluatedMines = payload.value("evaluated_mines", json::array());
    const json &tracks = payload.value("tracks", json::array());
    const json &timeline = payload.value("timeline", json::array());
    
    summary["total_usvs"] = tings.size();
    summary["total_suspect_mines"] = suspectMines.size();
    summary["total_confirmed_mines"] = confirmedMines.size();
    summary["total_cleared_mines"] = clearedMines.size();
    summary["total_destroyed_mines"] = destroyedMines.size();
    summary["total_evaluated_mines"] = evaluatedMines.size();
    summary["total_tracks"] = tracks.size();
    summary["total_events"] = timeline.size();
    result["summary"] = summary;
    
    // USV统计
    json usvStats = json::array();
    for (size_t i = 0; i < tings.size(); ++i) {
        const json &ting = tings[i];
        std::string tingId = ting.value("id", "");
        
        // 计算该USV的轨迹
        std::vector<json> tingTracks;
        for (const auto &track : tracks) {
            if (track.value("ting_id", "") == tingId) {
                tingTracks.push_back(track);
            }
        }
        
        // 计算移动距离
        double totalDistance = 0.0;
        if (tingTracks.size() > 1) {
            for (size_t j = 0; j < tingTracks.size() - 1; ++j) {
                if (tingTracks[j].contains("position") && tingTracks[j + 1].contains("position")) {
                    totalDistance += calculateDistance(tingTracks[j]["position"], tingTracks[j + 1]["position"]);
                }
            }
        }
        
        // 计算移动时间
        double moveTime = 0.0;
        if (!tingTracks.empty()) {
            double firstTime = tingTracks[0].value("timestamp", 0.0);
            double lastTime = tingTracks.back().value("timestamp", 0.0);
            if (lastTime > firstTime) {
                moveTime = lastTime - firstTime;
            }
        }
        
        // 计算平均速度
        double avgSpeed = 0.0;
        if (moveTime > 0) {
            avgSpeed = totalDistance / moveTime;
        }
        
        json usvStat;
        usvStat["id"] = tingId;
        usvStat["name"] = ting.value("name", "");
        usvStat["track_points"] = static_cast<int>(tingTracks.size());
        usvStat["total_distance_m"] = round(totalDistance * 100) / 100.0;
        usvStat["move_time_s"] = round(moveTime * 100) / 100.0;
        usvStat["avg_speed_mps"] = round(avgSpeed * 100) / 100.0;
        usvStat["speed_mps"] = ting.value("speed_mps", 0.0);
        usvStat["sonar_range_m"] = ting.value("sonar_range_m", 0.0);
        usvStats.push_back(usvStat);
    }
    result["usv_stats"] = usvStats;
    
    // 水雷统计
    json mineStats;
    int totalDiscovered = suspectMines.size() + confirmedMines.size();
    mineStats["discovery_rate"] = 0.0;
    mineStats["confirmation_rate"] = 0.0;
    mineStats["destruction_rate"] = 0.0;
    mineStats["evaluation_rate"] = 0.0;
    
    if (totalDiscovered > 0) {
        mineStats["confirmation_rate"] = round((confirmedMines.size() * 100.0 / totalDiscovered) * 100) / 100.0;
        mineStats["destruction_rate"] = round((destroyedMines.size() * 100.0 / totalDiscovered) * 100) / 100.0;
        if (destroyedMines.size() > 0) {
            mineStats["evaluation_rate"] = round((evaluatedMines.size() * 100.0 / destroyedMines.size()) * 100) / 100.0;
        }
    }
    result["mine_stats"] = mineStats;
    
    // 时间统计
    json timeStats;
    double createdAt = payload.value("created_at", 0.0);
    double updatedAt = payload.value("updated_at", 0.0);
    timeStats["total_duration_s"] = 0.0;
    if (createdAt > 0 && updatedAt > createdAt) {
        timeStats["total_duration_s"] = round((updatedAt - createdAt) * 100) / 100.0;
    }
    
    // 从时间线计算各阶段耗时
    json stageDurations;
    std::map<std::string, std::vector<double>> stageTimes;
    for (const auto &event : timeline) {
        std::string stage = event.value("stage", "unknown");
        double timestamp = event.value("timestamp", 0.0);
        if (timestamp > 0) {
            stageTimes[stage].push_back(timestamp);
        }
    }
    for (const auto &pair : stageTimes) {
        if (pair.second.size() >= 2) {
            double minTime = *std::min_element(pair.second.begin(), pair.second.end());
            double maxTime = *std::max_element(pair.second.begin(), pair.second.end());
            stageDurations[pair.first] = round((maxTime - minTime) * 100) / 100.0;
        }
    }
    timeStats["stage_duration_s"] = stageDurations;
    result["time_stats"] = timeStats;
    
    // 效率统计
    json efficiency;
    efficiency["mines_per_usv"] = 0.0;
    efficiency["distance_per_mine"] = 0.0;
    efficiency["time_per_mine"] = 0.0;
    
    if (tings.size() > 0) {
        efficiency["mines_per_usv"] = round((totalDiscovered * 100.0 / tings.size()) * 100) / 100.0;
    }
    
    double totalDistanceAll = 0.0;
    for (const auto &usvStat : usvStats) {
        totalDistanceAll += usvStat.value("total_distance_m", 0.0);
    }
    
    if (totalDiscovered > 0) {
        efficiency["distance_per_mine"] = round((totalDistanceAll * 100.0 / totalDiscovered) * 100) / 100.0;
        if (timeStats["total_duration_s"].get<double>() > 0) {
            efficiency["time_per_mine"] = round((timeStats["total_duration_s"].get<double>() * 100.0 / totalDiscovered) * 100) / 100.0;
        }
    }
    result["efficiency"] = efficiency;
    
    return result;
}

int main(int argc, char *argv[])
{
    QCoreApplication app(argc, argv);

    const int port = loadPortFromMeta(4102);
    httplib::Server server;

    server.Post("/analyze", [](const httplib::Request &req, httplib::Response &res) {
        try {
            if (req.body.empty()) {
                throw std::runtime_error("请求体不能为空");
            }
            const json payload = json::parse(req.body);
            const json result = analyzeTask(payload);
            res.set_content(result.dump(), "application/json");
            res.status = 200;
            std::cout << "[FSL_Statistics] 分析完成，任务ID: " << result.value("task_id", "") << std::endl;
        } catch (const std::exception &ex) {
            json error;
            error["error"] = ex.what();
            res.set_content(error.dump(), "application/json");
            res.status = 400;
            std::cerr << "[FSL_Statistics] 请求错误: " << ex.what() << std::endl;
        } catch (...) {
            json error;
            error["error"] = "未知错误";
            res.set_content(error.dump(), "application/json");
            res.status = 500;
            std::cerr << "[FSL_Statistics] 未知错误" << std::endl;
        }
    });

    server.Get("/healthz", [](const httplib::Request &, httplib::Response &res) {
        json ok;
        ok["status"] = "ok";
        res.set_content(ok.dump(), "application/json");
    });

    std::cout << "[FSL_Statistics] 服务启动，监听 0.0.0.0:" << port << std::endl;
    server.listen("0.0.0.0", port);
    return 0;
}

