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
#include <random>
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
            qWarning() << "[SimTargetRecognize] 无法打开 meta.ini:" << path << "-" << file.errorString();
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

    qWarning() << "[SimTargetRecognize] 未在 meta.ini 中找到服务" << serviceName << "，使用默认端口" << defaultPort;
    return defaultPort;
}

class TargetRecognizeServer : public QRunnable
{
public:
    explicit TargetRecognizeServer(int listenPort)
        : port(listenPort)
    {
    }

    virtual void run()
    {
        httplib::Server svr;

        // recognizeTarget 服务：使用 HTTP POST，接收声呐图像路径
        svr.Post("/recognizeTarget", [](const httplib::Request& req, httplib::Response& res) {
            std::cout << "[SimTargetRecognize] 收到 /recognizeTarget 请求" << std::endl;

            try
            {
                // 从 request body 中获取 JSON 参数
                if (req.body.empty()) {
                    json err = {{"success", false}, {"error", "Empty request body"}};
                    res.set_content(err.dump(), "application/json");
                    res.status = 400;
                    std::cout << "[SimTargetRecognize] /recognizeTarget 响应（错误）: " << err.dump() << std::endl;
                    return;
                }

                std::cout << "[SimTargetRecognize] 请求体: " << req.body << std::endl;
                auto input = json::parse(req.body);

                // 打印请求数据
                std::cout << "[SimTargetRecognize] /recognizeTarget 输入数据:" << std::endl;
                std::cout << input.dump(2) << std::endl;

                // 解析输入：图像路径
                std::string imagePath = input.value("image_path", "");
                if (imagePath.empty()) {
                    json err = {{"success", false}, {"error", "图像路径不能为空"}};
                    res.set_content(err.dump(), "application/json");
                    res.status = 400;
                    std::cout << "[SimTargetRecognize] /recognizeTarget 响应（错误）: " << err.dump() << std::endl;
                    return;
                }

                std::cout << "[SimTargetRecognize] 图像路径: " << imagePath << std::endl;

                // 模拟目标识别处理延迟（2秒）
                std::cout << "[SimTargetRecognize] 开始目标识别，预计耗时 2 秒..." << std::endl;
                std::this_thread::sleep_for(std::chrono::milliseconds(2000));
                std::cout << "[SimTargetRecognize] 目标识别完成" << std::endl;

                // 模拟目标识别
                // 在实际应用中，这里会：
                // 1. 读取图像文件
                // 2. 使用深度学习模型进行目标识别
                // 3. 分析图像特征
                // 4. 返回识别结果

                // 使用随机数生成器模拟识别结果
                std::random_device rd;
                std::mt19937 gen(rd());
                std::vector<std::string> targetTypes = {"水雷", "蛙人", "UUV", "潜艇", "水面舰艇", "未知目标"};
                std::uniform_int_distribution<> type_dist(0, targetTypes.size() - 1);
                std::vector<std::string> targetSizes = {"小", "中", "大"};
                std::uniform_int_distribution<> size_dist(0, targetSizes.size() - 1);
                std::uniform_real_distribution<> confidence_dist(0.7, 0.99);

                std::string recognizedType = targetTypes[type_dist(gen)];
                std::string targetSize = targetSizes[size_dist(gen)];
                double confidence = std::round(confidence_dist(gen) * 100) / 100.0;

                std::cout << "[SimTargetRecognize] 识别结果: " << recognizedType << " (尺寸: " << targetSize << ", 置信度: " << confidence << ")" << std::endl;

                // 构建响应
                json result;
                result["success"] = true;
                result["message"] = "目标识别成功";
                result["image_path"] = imagePath;
                result["target_type"] = recognizedType;
                result["size"] = targetSize;
                result["confidence"] = confidence;
                result["recognize_time"] = std::time(nullptr);
                
                std::string responseStr = result.dump();
                res.set_content(responseStr, "application/json");
                res.status = 200;

                std::cout << "[SimTargetRecognize] /recognizeTarget 响应:" << std::endl;
                std::cout << result.dump(2) << std::endl;
            }
            catch (std::exception& e)
            {
                json err = {{"success", false}, {"error", std::string("Parse error: ") + e.what()}};
                res.set_content(err.dump(), "application/json");
                res.status = 500;
                std::cout << "[SimTargetRecognize] /recognizeTarget 响应（错误）: " << err.dump() << std::endl;
            }
        });

        std::cout << "========================================" << std::endl;
        std::cout << "  SimTargetRecognize 服务器已启动" << std::endl;
        std::cout << "========================================" << std::endl;
        std::cout << "可用端点:" << std::endl;
        std::cout << "  - POST /recognizeTarget (JSON 请求体)" << std::endl;
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

    const int port = loadPortFromMeta(QStringLiteral("recognizeTarget"), 3500);
    TargetRecognizeServer *server = new TargetRecognizeServer(port);
    QThreadPool::globalInstance()->start(server);

    return app.exec();
}

