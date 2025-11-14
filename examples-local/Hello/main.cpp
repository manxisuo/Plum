#include <QCoreApplication>
#include <QThreadPool>
#include <QRunnable>
#include <QDebug>

#include <iostream>
#include <thread>

#include "httplib.h"
#include "json.hpp"

using json = nlohmann::json;

class Task : public QRunnable
{
public:
    virtual void run()
    {
        httplib::Server svr;

        // body: {taskId,name,payload}
        svr.Post("/task001", [&](const httplib::Request& req, httplib::Response& res) {
            std::cout << "[worker] /task001 request: " << req.body << std::endl;

            try
            {
                auto j = json::parse(req.body);
                std::string name = j.value("name", "");
                json payload = j.value("payload", json::object());

                json result = {{"ok", true}, {"msg", "task001 has received your request."}};
                res.set_content(result.dump(), "application/json");
                res.status = 200;

            }
            catch (std::exception& e)
            {
                json err = {{"ok", false}, {"error", std::string("bad json: ") + e.what()}};
                res.set_content(err.dump(), "application/json");
                res.status = 400;
            }
        });

        svr.listen("0.0.0.0", 9111);
    }
};

int main(int argc, char *argv[])
{
    QCoreApplication a(argc, argv);

    QThreadPool::globalInstance()->start(new Task);

    qDebug() << "App Hello is running...";

    return a.exec();
}
