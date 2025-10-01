#pragma once
#include <string>

bool ensure_dir(const std::string& path);
bool file_exists(const std::string& path);
bool unzip_zip(const std::string& zip_path, const std::string& out_dir);


