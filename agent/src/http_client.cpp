#include "http_client.hpp"
#include <curl/curl.h>
#include <stdexcept>

namespace {
static size_t write_callback(char* ptr, size_t size, size_t nmemb, void* userdata) {
	auto* out = reinterpret_cast<std::string*>(userdata);
	out->append(ptr, size * nmemb);
	return size * nmemb;
}
}

HttpClient::HttpClient() {
	curl_global_init(CURL_GLOBAL_DEFAULT);
	curl_ = curl_easy_init();
}

HttpClient::~HttpClient() {
	if (curl_) {
		curl_easy_cleanup(static_cast<CURL*>(curl_));
	}
	curl_global_cleanup();
}

HttpResponse HttpClient::post_json(const std::string& url, const std::string& json_body, int timeout_sec) {
	if (!curl_) return {};
	std::string response;
	char errbuf[CURL_ERROR_SIZE] = {0};
	struct curl_slist* headers = nullptr;
	headers = curl_slist_append(headers, "Content-Type: application/json");

	CURL* c = static_cast<CURL*>(curl_);
	curl_easy_reset(c);
	curl_easy_setopt(c, CURLOPT_NOSIGNAL, 1L);
	curl_easy_setopt(c, CURLOPT_ERRORBUFFER, errbuf);
	curl_easy_setopt(c, CURLOPT_URL, url.c_str());
	curl_easy_setopt(c, CURLOPT_POST, 1L);
	curl_easy_setopt(c, CURLOPT_HTTPHEADER, headers);
	curl_easy_setopt(c, CURLOPT_POSTFIELDS, json_body.c_str());
	curl_easy_setopt(c, CURLOPT_POSTFIELDSIZE, json_body.size());
	curl_easy_setopt(c, CURLOPT_TIMEOUT, timeout_sec);
	curl_easy_setopt(c, CURLOPT_WRITEFUNCTION, write_callback);
	curl_easy_setopt(c, CURLOPT_WRITEDATA, &response);

	CURLcode res = curl_easy_perform(c);
	long status = 0;
	curl_easy_getinfo(c, CURLINFO_RESPONSE_CODE, &status);
	// 清理并解除 header 绑定以避免悬空指针跨请求复用
	curl_easy_setopt(c, CURLOPT_HTTPHEADER, nullptr);
	curl_slist_free_all(headers);

	if (res != CURLE_OK) {
		return {0, {}};
	}
	return {status, response};
}

HttpResponse HttpClient::get(const std::string& url, int timeout_sec) {
	if (!curl_) return {};
	std::string response;
	char errbuf[CURL_ERROR_SIZE] = {0};
	CURL* c = static_cast<CURL*>(curl_);
	curl_easy_reset(c);
	curl_easy_setopt(c, CURLOPT_NOSIGNAL, 1L);
	curl_easy_setopt(c, CURLOPT_ERRORBUFFER, errbuf);
	curl_easy_setopt(c, CURLOPT_URL, url.c_str());
	curl_easy_setopt(c, CURLOPT_HTTPGET, 1L);
	curl_easy_setopt(c, CURLOPT_TIMEOUT, timeout_sec);
	curl_easy_setopt(c, CURLOPT_WRITEFUNCTION, write_callback);
	curl_easy_setopt(c, CURLOPT_WRITEDATA, &response);
	CURLcode res = curl_easy_perform(c);
	long status = 0;
	curl_easy_getinfo(c, CURLINFO_RESPONSE_CODE, &status);
	if (res != CURLE_OK) {
		return {0, {}};
	}
	return {status, response};
}


