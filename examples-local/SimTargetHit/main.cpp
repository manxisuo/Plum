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
#include <ctime>

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
            qWarning() << "[SimTargetHit] 无法打开 meta.ini:" << path << "-" << file.errorString();
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

    qWarning() << "[SimTargetHit] 未在 meta.ini 中找到服务" << serviceName << "，使用默认端口" << defaultPort;
    return defaultPort;
}

class TargetHitServer : public QRunnable
{
public:
    explicit TargetHitServer(int listenPort)
        : port(listenPort)
    {
    }

    virtual void run()
    {
        httplib::Server svr;

        // hitTarget 服务：使用 HTTP POST，接收目标 id 和经纬度
        svr.Post("/hitTarget", [](const httplib::Request& req, httplib::Response& res) {
            std::cout << "[SimTargetHit] 收到 /hitTarget 请求" << std::endl;

            try
            {
                // 从 request body 中获取 JSON 参数
                if (req.body.empty()) {
                    json err = {{"success", false}, {"error", "Empty request body"}};
                    res.set_content(err.dump(), "application/json");
                    res.status = 400;
                    std::cout << "[SimTargetHit] /hitTarget 响应（错误）: " << err.dump() << std::endl;
                    return;
                }

                std::cout << "[SimTargetHit] 请求体: " << req.body << std::endl;
                auto input = json::parse(req.body);

                // 打印请求数据
                std::cout << "[SimTargetHit] /hitTarget 输入数据:" << std::endl;
                std::cout << input.dump(2) << std::endl;

                // 解析输入：目标 id 和经纬度
                int targetId = input.value("id", 0);
                double longitude = input.value("longitude", 0.0);
                double latitude = input.value("latitude", 0.0);

                // 验证输入
                if (targetId <= 0) {
                    json err = {{"success", false}, {"error", "目标 ID 无效"}};
                    res.set_content(err.dump(), "application/json");
                    res.status = 400;
                    std::cout << "[SimTargetHit] /hitTarget 响应（错误）: " << err.dump() << std::endl;
                    return;
                }

                if (longitude == 0.0 && latitude == 0.0) {
                    json err = {{"success", false}, {"error", "经纬度无效"}};
                    res.set_content(err.dump(), "application/json");
                    res.status = 400;
                    std::cout << "[SimTargetHit] /hitTarget 响应（错误）: " << err.dump() << std::endl;
                    return;
                }

                std::cout << "[SimTargetHit] 目标 ID: " << targetId << std::endl;
                std::cout << "[SimTargetHit] 目标位置: (" << longitude << ", " << latitude << ")" << std::endl;

                // 模拟目标打击处理延迟（2秒）
                std::cout << "[SimTargetHit] 开始目标打击，预计耗时 2 秒..." << std::endl;
                std::this_thread::sleep_for(std::chrono::milliseconds(2000));
                std::cout << "[SimTargetHit] 目标打击完成" << std::endl;

                // 模拟目标打击
                // 在实际应用中，这里会：
                // 1. 计算打击参数
                // 2. 发送打击指令到武器系统
                // 3. 监控打击结果
                // 4. 评估打击效果

                // 构建响应
                json result;
                result["success"] = true;
                result["message"] = "目标打击成功";
                result["target_id"] = targetId;
                result["longitude"] = longitude;
                result["latitude"] = latitude;
                result["hit_time"] = std::time(nullptr);
                result["damage"] = "高";
                result["status"] = "destroyed";
                
                std::string responseStr = result.dump();
                res.set_content(responseStr, "application/json");
                res.status = 200;

                std::cout << "[SimTargetHit] /hitTarget 响应:" << std::endl;
                std::cout << result.dump(2) << std::endl;
            }
            catch (std::exception& e)
            {
                json err = {{"success", false}, {"error", std::string("Parse error: ") + e.what()}};
                res.set_content(err.dump(), "application/json");
                res.status = 500;
                std::cout << "[SimTargetHit] /hitTarget 响应（错误）: " << err.dump() << std::endl;
            }
        });

        std::cout << "========================================" << std::endl;
        std::cout << "  SimTargetHit 服务器已启动" << std::endl;
        std::cout << "========================================" << std::endl;
        std::cout << "可用端点:" << std::endl;
        std::cout << "  - POST /hitTarget (JSON 请求体)" << std::endl;
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

    const int port = loadPortFromMeta(QStringLiteral("hitTarget"), 3400);
    TargetHitServer *server = new TargetHitServer(port);
    QThreadPool::globalInstance()->start(server);

    return app.exec();
}

