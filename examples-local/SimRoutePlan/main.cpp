#include <QCoreApplication>
#include <QThreadPool>
#include <QRunnable>
#include <QDebug>
#include <QFile>
#include <QTextStream>
#include <QStringList>

#include <iostream>
#include <thread>
#include <vector>
#include <chrono>

#include "httplib.h"
#include "json.hpp"

using json = nlohmann::json;

static int loadPortFromMeta(const QString &serviceName, int defaultPort)
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
            qWarning() << "[SimRoutePlan] 无法打开 meta.ini:" << path << "-" << file.errorString();
            continue;
        }
        QTextStream in(&file);
        while (!in.atEnd()) {
            QString line = in.readLine().trimmed();
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
            if (parts[0].trimmed() == serviceName) {
                bool ok = false;
                const int port = parts[2].trimmed().toInt(&ok);
                if (ok) {
                    return port;
                }
            }
        }
    }

    qWarning() << "[SimRoutePlan] 未在 meta.ini 中找到服务" << serviceName << "，使用默认端口" << defaultPort;
    return defaultPort;
}

class RoutePlanServer : public QRunnable
{
public:
    explicit RoutePlanServer(int listenPort)
        : port(listenPort)
    {
    }

    virtual void run()
    {
        httplib::Server svr;

        // planRoute1 服务：使用 HTTP POST，参数通过 request body 传递（JSON 格式）
        svr.Post("/planRoute1", [](const httplib::Request& req, httplib::Response& res) {
            std::cout << "[SimRoutePlan] 收到 /planRoute1 请求" << std::endl;

            try
            {
                // 从 request body 中获取 JSON 参数
                if (req.body.empty()) {
                    json err = {{"success", false}, {"error", "Empty request body"}};
                    res.set_content(err.dump(), "application/json");
                    res.status = 400;
                    std::cout << "[SimRoutePlan] /planRoute1 响应（错误）: " << err.dump() << std::endl;
                    return;
                }

                std::cout << "[SimRoutePlan] 请求体: " << req.body << std::endl;
                auto input = json::parse(req.body);

                // 打印请求数据
                std::cout << "[SimRoutePlan] /planRoute1 输入数据:" << std::endl;
                std::cout << input.dump(2) << std::endl;

                // 解析输入：两个点（经纬度）和一个障碍物（多边形）
                json point1 = input.value("point1", json::object());
                json point2 = input.value("point2", json::object());
                json obstacle = input.value("obstacle", json::object());

                double lon1 = point1.value("longitude", 0.0);
                double lat1 = point1.value("latitude", 0.0);
                double lon2 = point2.value("longitude", 0.0);
                double lat2 = point2.value("latitude", 0.0);
                json obstaclePolygon = obstacle.value("polygon", json::array());

                std::cout << "[SimRoutePlan] 点 1: (" << lon1 << ", " << lat1 << ")" << std::endl;
                std::cout << "[SimRoutePlan] 点 2: (" << lon2 << ", " << lat2 << ")" << std::endl;
                std::cout << "[SimRoutePlan] 障碍物多边形有 " << obstaclePolygon.size() << " 个点" << std::endl;

                // 模拟航路规划处理延迟（2秒）
                std::cout << "[SimRoutePlan] 开始航路规划，预计耗时 2 秒..." << std::endl;
                std::this_thread::sleep_for(std::chrono::milliseconds(2000));
                std::cout << "[SimRoutePlan] 航路规划完成" << std::endl;

                // 模拟航路规划算法（planRoute1）
                // 生成一个简单的航路：起点 -> 中间点 -> 终点
                json route = json::array();

                // 起点
                json startPoint = json::object();
                startPoint["longitude"] = lon1;
                startPoint["latitude"] = lat1;
                route.push_back(startPoint);

                // 中间点（模拟绕障碍物）
                json midPoint = json::object();
                midPoint["longitude"] = (lon1 + lon2) / 2.0 + 0.001;  // 稍微偏移
                midPoint["latitude"] = (lat1 + lat2) / 2.0 + 0.001;
                route.push_back(midPoint);

                // 终点
                json endPoint = json::object();
                endPoint["longitude"] = lon2;
                endPoint["latitude"] = lat2;
                route.push_back(endPoint);

                // 构建响应
                json result;
                result["success"] = true;
                result["algorithm"] = "planRoute1";
                result["route"] = route;

                std::string responseStr = result.dump();
                res.set_content(responseStr, "application/json");
                res.status = 200;

                // 打印响应
                std::cout << "[SimRoutePlan] /planRoute1 响应:" << std::endl;
                std::cout << result.dump(2) << std::endl;
            }
            catch (std::exception& e)
            {
                json err = {{"success", false}, {"error", std::string("Error: ") + e.what()}};
                res.set_content(err.dump(), "application/json");
                res.status = 400;
                std::cout << "[SimRoutePlan] /planRoute1 响应（错误）: " << err.dump() << std::endl;
            }
        });

        // planRoute2 服务：使用 HTTP POST，参数通过 request body 传递（JSON 格式）
        svr.Post("/planRoute2", [](const httplib::Request& req, httplib::Response& res) {
            std::cout << "[SimRoutePlan] 收到 /planRoute2 请求" << std::endl;

            try
            {
                // 从 request body 中获取 JSON 参数
                if (req.body.empty()) {
                    json err = {{"success", false}, {"error", "Empty request body"}};
                    res.set_content(err.dump(), "application/json");
                    res.status = 400;
                    std::cout << "[SimRoutePlan] /planRoute2 响应（错误）: " << err.dump() << std::endl;
                    return;
                }

                std::cout << "[SimRoutePlan] 请求体: " << req.body << std::endl;
                auto input = json::parse(req.body);

                // 打印请求数据
                std::cout << "[SimRoutePlan] /planRoute2 输入数据:" << std::endl;
                std::cout << input.dump(2) << std::endl;

                // 解析输入：两个点（经纬度）和一个障碍物（多边形）
                json point1 = input.value("point1", json::object());
                json point2 = input.value("point2", json::object());
                json obstacle = input.value("obstacle", json::object());

                double lon1 = point1.value("longitude", 0.0);
                double lat1 = point1.value("latitude", 0.0);
                double lon2 = point2.value("longitude", 0.0);
                double lat2 = point2.value("latitude", 0.0);
                json obstaclePolygon = obstacle.value("polygon", json::array());

                std::cout << "[SimRoutePlan] 点 1: (" << lon1 << ", " << lat1 << ")" << std::endl;
                std::cout << "[SimRoutePlan] 点 2: (" << lon2 << ", " << lat2 << ")" << std::endl;
                std::cout << "[SimRoutePlan] 障碍物多边形有 " << obstaclePolygon.size() << " 个点" << std::endl;

                // 模拟航路规划处理延迟（2秒）
                std::cout << "[SimRoutePlan] 开始航路规划，预计耗时 2 秒..." << std::endl;
                std::this_thread::sleep_for(std::chrono::milliseconds(2000));
                std::cout << "[SimRoutePlan] 航路规划完成" << std::endl;

                // 模拟航路规划算法（planRoute2）- 不同的算法，生成更多中间点
                json route = json::array();

                // 起点
                json startPoint = json::object();
                startPoint["longitude"] = lon1;
                startPoint["latitude"] = lat1;
                route.push_back(startPoint);

                // 多个中间点（模拟更复杂的路径规划）
                for (int i = 1; i <= 3; i++) {
                    double ratio = i / 4.0;
                    json midPoint = json::object();
                    midPoint["longitude"] = lon1 + (lon2 - lon1) * ratio + 0.0005 * i;
                    midPoint["latitude"] = lat1 + (lat2 - lat1) * ratio + 0.0005 * i;
                    route.push_back(midPoint);
                }

                // 终点
                json endPoint = json::object();
                endPoint["longitude"] = lon2;
                endPoint["latitude"] = lat2;
                route.push_back(endPoint);

                // 构建响应
                json result;
                result["success"] = true;
                result["algorithm"] = "planRoute2";
                result["route"] = route;

                std::string responseStr = result.dump();
                res.set_content(responseStr, "application/json");
                res.status = 200;

                // 打印响应
                std::cout << "[SimRoutePlan] /planRoute2 响应:" << std::endl;
                std::cout << result.dump(2) << std::endl;
            }
            catch (std::exception& e)
            {
                json err = {{"success", false}, {"error", std::string("Error: ") + e.what()}};
                res.set_content(err.dump(), "application/json");
                res.status = 400;
                std::cout << "[SimRoutePlan] /planRoute2 响应（错误）: " << err.dump() << std::endl;
            }
        });

        std::cout << "[SimRoutePlan] 服务器正在 0.0.0.0:" << port << " 启动" << std::endl;
        std::cout << "[SimRoutePlan] 可用端点:" << std::endl;
        std::cout << "  - POST /planRoute1 (JSON 请求体)" << std::endl;
        std::cout << "  - POST /planRoute2 (JSON 请求体)" << std::endl;

        svr.listen("0.0.0.0", port);
    }

private:
    int port;
};

int main(int argc, char *argv[])
{
    QCoreApplication a(argc, argv);

    const int port = loadPortFromMeta(QStringLiteral("planRoute1"), 3100);
    QThreadPool::globalInstance()->start(new RoutePlanServer(port));

    qDebug() << "SimRoutePlan 应用正在运行...";

    return a.exec();
}
