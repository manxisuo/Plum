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
#include <random>
#include <ctime>
#include <chrono>
#include <cstdio>

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
            qWarning() << "[SimSonar] 无法打开 meta.ini:" << path << "-" << file.errorString();
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

    qWarning() << "[SimSonar] 未在 meta.ini 中找到服务" << serviceName << "，使用默认端口" << defaultPort;
    return defaultPort;
}

class SonarServer : public QRunnable
{
public:
    explicit SonarServer(int listenPort)
        : port(listenPort)
    {
    }

    virtual void run()
    {
        httplib::Server svr;

        // detectTarget 服务：使用 HTTP GET，不需要参数，返回目标信息
        svr.Get("/detectTarget", [](const httplib::Request& req, httplib::Response& res) {
            std::cout << "[SimSonar] 收到 /detectTarget 请求" << std::endl;

            try
            {
                // 模拟生成目标信息
                // 在实际应用中，这里会：
                // 1. 从声纳设备读取数据
                // 2. 进行目标识别和分类
                // 3. 计算目标位置（经纬度）
                // 4. 评估目标尺寸

                // 使用随机数生成器
                std::random_device rd;
                std::mt19937 gen(rd());
                
                // 模拟探测延迟（3-5秒）
                std::uniform_int_distribution<> delay_dist(3000, 5000);  // 3-5秒（毫秒）
                int delay_ms = delay_dist(gen);
                
                std::cout << "[SimSonar] 开始目标探测，预计耗时 " << (delay_ms / 1000.0) << " 秒..." << std::endl;
                std::this_thread::sleep_for(std::chrono::milliseconds(delay_ms));
                std::cout << "[SimSonar] 目标探测完成" << std::endl;

                // 使用随机数生成器模拟目标检测
                std::uniform_real_distribution<> lon_dist(116.0, 116.5);   // 经度范围
                std::uniform_real_distribution<> lat_dist(39.0, 39.5);     // 纬度范围
                
                // 目标类型列表
                std::vector<std::string> targetTypes = {"水雷", "蛙人", "UUV", "潜艇", "水面舰艇", "未知目标"};
                std::uniform_int_distribution<> type_dist(0, targetTypes.size() - 1);
                
                // 目标尺寸列表
                std::vector<std::string> targetSizes = {"小", "中", "大"};
                std::uniform_int_distribution<> size_dist(0, targetSizes.size() - 1);
                
                // 生成2-3个随机目标
                std::uniform_int_distribution<> target_count_dist(2, 3);
                int targetCount = target_count_dist(gen);

                json targets = json::array();
                
                for (int i = 0; i < targetCount; i++) {
                    json target;
                    target["id"] = i + 1;
                    target["longitude"] = lon_dist(gen);
                    target["latitude"] = lat_dist(gen);
                    
                    // 添加距离（米，模拟）
                    std::uniform_real_distribution<> distance_dist(50, 5000);
                    target["distance"] = std::round(distance_dist(gen));
                    
                    // 添加图像路径（模拟，格式：images/sonar_image_001.jpg）
                    char imagePathBuf[256];
                    snprintf(imagePathBuf, sizeof(imagePathBuf), "images/sonar_image_%03d.jpg", i + 1);
                    std::string imagePath = std::string(imagePathBuf);
                    target["image_path"] = imagePath;
                    
                    targets.push_back(target);
                    
                    std::cout << "[SimSonar] 检测到目标 " << (i + 1) << ": "
                              << "位置=(" << target["longitude"] << ", " << target["latitude"] << "), "
                              << "距离=" << target["distance"] << "m, "
                              << "图像=" << imagePath << std::endl;
                }

                // 构建响应
                json result;
                result["success"] = true;
                result["message"] = "目标检测完成";
                result["target_count"] = targetCount;
                result["targets"] = targets;
                result["timestamp"] = std::time(nullptr);

                std::string responseStr = result.dump();
                res.set_content(responseStr, "application/json");
                res.status = 200;

                std::cout << "[SimSonar] /detectTarget 响应:" << std::endl;
                std::cout << result.dump(2) << std::endl;
            }
            catch (std::exception& e)
            {
                json err = {{"success", false}, {"error", std::string("检测错误: ") + e.what()}};
                res.set_content(err.dump(), "application/json");
                res.status = 500;
                std::cout << "[SimSonar] /detectTarget 响应（错误）: " << err.dump() << std::endl;
            }
        });

        std::cout << "========================================" << std::endl;
        std::cout << "  SimSonar 服务器已启动" << std::endl;
        std::cout << "========================================" << std::endl;
        std::cout << "可用端点:" << std::endl;
        std::cout << "  - GET /detectTarget (无需参数)" << std::endl;
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

    const int port = loadPortFromMeta(QStringLiteral("detectTarget"), 3300);
    SonarServer *server = new SonarServer(port);
    QThreadPool::globalInstance()->start(server);

    return app.exec();
}
