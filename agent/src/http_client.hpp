#pragma once

#include <string>
#include <vector>

struct HttpResponse {
	long status_code {0};
	std::string body;
};

class HttpClient {
public:
	HttpClient();
	~HttpClient();

	HttpResponse post_json(const std::string& url, const std::string& json_body, int timeout_sec = 5);
	HttpResponse get(const std::string& url, int timeout_sec = 5);

private:
	void* curl_ {nullptr};
};


