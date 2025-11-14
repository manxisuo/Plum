#include <QCoreApplication>
#include <QFile>
#include <QTextStream>
#include <QDebug>
#include <QStringList>

#include "../FSL_Common/httplib.h"
#include "../FSL_Common/json.hpp"

#include <iostream>
#include <stdexcept>
#include <vector>

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
            qWarning() << "[FSL_Plan] 无法打开 meta.ini:" << path << "-" << file.errorString();
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
            if (parts[0].trimmed() == "planArea") {
                bool ok = false;
                const int port = parts[2].trimmed().toInt(&ok);
                if (ok) {
                    return port;
                }
            }
        }
    }

    qWarning() << "[FSL_Plan] 未在 meta.ini 中找到 planArea 服务端口，使用默认端口" << defaultPort;
    return defaultPort;
}

static json buildWorkZones(const json &payload)
{
    if (!payload.contains("task_area")) {
        throw std::runtime_error("缺少 task_area 字段");
    }
    const json &taskArea = payload.at("task_area");
    if (!taskArea.contains("top_left") || !taskArea.contains("bottom_right")) {
        throw std::runtime_error("task_area 需包含 top_left 与 bottom_right");
    }

    const json &topLeft = taskArea.at("top_left");
    const json &bottomRight = taskArea.at("bottom_right");

    const double topLat = topLeft.value("lat", 0.0);
    const double leftLon = topLeft.value("lon", 0.0);
    const double bottomLat = bottomRight.value("lat", 0.0);
    const double rightLon = bottomRight.value("lon", 0.0);

    if (rightLon <= leftLon) {
        throw std::runtime_error("矩形经度范围无效：右下角经度必须大于左上角经度");
    }
    if (topLat <= bottomLat) {
        throw std::runtime_error("矩形纬度范围无效：左上角纬度必须大于右下角纬度");
    }

    int tingCount = payload.value("ting_count", 4);
    if (tingCount <= 0) {
        throw std::runtime_error("ting_count 必须为正整数");
    }

    const double totalWidth = rightLon - leftLon;
    const double step = totalWidth / tingCount;

    json zones = json::array();
    for (int i = 0; i < tingCount; ++i) {
        const double zoneLeft = leftLon + step * i;
        double zoneRight = zoneLeft + step;
        if (i == tingCount - 1) {
            zoneRight = rightLon;
        }
        json zone;
        zone["id"] = QString("zone-%1").arg(i + 1).toStdString();
        zone["index"] = i;
        zone["top_left"] = {{"lat", topLat}, {"lon", zoneLeft}};
        zone["bottom_right"] = {{"lat", bottomLat}, {"lon", zoneRight}};
        zones.push_back(zone);
    }

    json response;
    response["work_zones"] = zones;
    response["summary"] = {
        {"ting_count", tingCount},
        {"task_area", {
             {"top_left", topLeft},
             {"bottom_right", bottomRight}
         }}
    };

    return response;
}

int main(int argc, char *argv[])
{
    QCoreApplication app(argc, argv);

    const int port = loadPortFromMeta(4100);
    httplib::Server server;

    server.Post("/planArea", [](const httplib::Request &req, httplib::Response &res) {
        try {
            if (req.body.empty()) {
                throw std::runtime_error("请求体不能为空");
            }
            const json payload = json::parse(req.body);
            const json result = buildWorkZones(payload);
            res.set_content(result.dump(), "application/json");
            res.status = 200;
            std::cout << "[FSL_Plan] 处理成功，返回 " << result["work_zones"].size() << " 个作业区" << std::endl;
        } catch (const std::exception &ex) {
            json error;
            error["error"] = ex.what();
            res.set_content(error.dump(), "application/json");
            res.status = 400;
            std::cerr << "[FSL_Plan] 请求错误: " << ex.what() << std::endl;
        } catch (...) {
            json error;
            error["error"] = "未知错误";
            res.set_content(error.dump(), "application/json");
            res.status = 500;
            std::cerr << "[FSL_Plan] 未知错误" << std::endl;
        }
    });

    server.Get("/healthz", [](const httplib::Request &, httplib::Response &res) {
        json ok;
        ok["status"] = "ok";
        res.set_content(ok.dump(), "application/json");
    });

    std::cout << "[FSL_Plan] 服务启动，监听 0.0.0.0:" << port << std::endl;
    server.listen("0.0.0.0", port);
    return 0;
}

