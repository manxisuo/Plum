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
            qWarning() << "[SimNaviControl] 无法打开 meta.ini:" << path << "-" << file.errorString();
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

    qWarning() << "[SimNaviControl] 未在 meta.ini 中找到服务" << serviceName << "，使用默认端口" << defaultPort;
    return defaultPort;
}

class NaviControlServer : public QRunnable
{
public:
    explicit NaviControlServer(int listenPort)
        : port(listenPort)
    {
    }

    virtual void run()
    {
        httplib::Server svr;

        // controlUSV 服务：使用 HTTP POST，接收路径（由 SimRoutePlan 返回的格式）
        svr.Post("/controlUSV", [](const httplib::Request& req, httplib::Response& res) {
            std::cout << "[SimNaviControl] 收到 /controlUSV 请求" << std::endl;

            try
            {
                // 从 request body 中获取 JSON 参数
                if (req.body.empty()) {
                    json err = {{"success", false}, {"error", "Empty request body"}};
                    res.set_content(err.dump(), "application/json");
                    res.status = 400;
                    std::cout << "[SimNaviControl] /controlUSV 响应（错误）: " << err.dump() << std::endl;
                    return;
                }

                std::cout << "[SimNaviControl] 请求体: " << req.body << std::endl;
                auto input = json::parse(req.body);

                // 打印请求数据
                std::cout << "[SimNaviControl] /controlUSV 输入数据:" << std::endl;
                std::cout << input.dump(2) << std::endl;

                // 解析输入：路径（route 数组，每个元素包含 longitude 和 latitude）
                json route = input.value("route", json::array());

                if (route.empty()) {
                    json err = {{"success", false}, {"error", "Route is empty"}};
                    res.set_content(err.dump(), "application/json");
                    res.status = 400;
                    std::cout << "[SimNaviControl] /controlUSV 响应（错误）: " << err.dump() << std::endl;
                    return;
                }

                std::cout << "[SimNaviControl] 收到路径，包含 " << route.size() << " 个航点" << std::endl;

                // 模拟 USV 导航控制处理延迟（2秒）
                std::cout << "[SimNaviControl] 开始航控启动，预计耗时 2 秒..." << std::endl;
                std::this_thread::sleep_for(std::chrono::milliseconds(2000));
                std::cout << "[SimNaviControl] 航控启动完成" << std::endl;

                // 模拟 USV 导航控制
                // 在实际应用中，这里会：
                // 1. 验证路径的有效性
                // 2. 发送控制指令到 USV 硬件
                // 3. 监控 USV 的航行状态
                // 4. 处理异常情况

                // 模拟处理过程
                bool navigationSuccess = true;
                std::string errorMessage = "";

                // 简单的路径验证：检查每个航点的格式
                for (size_t i = 0; i < route.size(); i++) {
                    json waypoint = route[i];
                    if (!waypoint.contains("longitude") || !waypoint.contains("latitude")) {
                        navigationSuccess = false;
                        errorMessage = "无效的航路点格式：" + std::to_string(i);
                        break;
                    }
                    double lon = waypoint.value("longitude", 0.0);
                    double lat = waypoint.value("latitude", 0.0);
                    std::cout << "[SimNaviControl] 航点 " << i << ": (" << lon << ", " << lat << ")" << std::endl;
                }

                // 构建响应
                json result;
                if (navigationSuccess) {
                    result["success"] = true;
                    result["message"] = "USV航控启动成功";
                    result["waypoints_count"] = route.size();
                    result["status"] = "navigating";
                    
                    std::string responseStr = result.dump();
                    res.set_content(responseStr, "application/json");
                    res.status = 200;

                    std::cout << "[SimNaviControl] /controlUSV 响应:" << std::endl;
                    std::cout << result.dump(2) << std::endl;
                } else {
                    result["success"] = false;
                    result["error"] = errorMessage;
                    
                    std::string responseStr = result.dump();
                    res.set_content(responseStr, "application/json");
                    res.status = 400;

                    std::cout << "[SimNaviControl] /controlUSV 响应（错误）: " << result.dump() << std::endl;
                }
            }
            catch (std::exception& e)
            {
                json err = {{"success", false}, {"error", std::string("Parse error: ") + e.what()}};
                res.set_content(err.dump(), "application/json");
                res.status = 500;
                std::cout << "[SimNaviControl] /controlUSV 响应（错误）: " << err.dump() << std::endl;
            }
        });

        std::cout << "========================================" << std::endl;
        std::cout << "  SimNaviControl 服务器已启动" << std::endl;
        std::cout << "========================================" << std::endl;
        std::cout << "可用端点:" << std::endl;
        std::cout << "  - POST /controlUSV (JSON 请求体)" << std::endl;
        std::cout << "监听地址: 0.0.0.0:" << port << std::endl;
        std::cout << "========================================" << std::endl;

        svr.listen("0.0.0.0", port);
    }

private:
    int port;
};

int main(int argc, char *argv[])
{
    QCoreApplication app(argc, argv);

    const int port = loadPortFromMeta(QStringLiteral("controlUSV"), 3200);
    NaviControlServer *server = new NaviControlServer(port);
    QThreadPool::globalInstance()->start(server);

    return app.exec();
}
