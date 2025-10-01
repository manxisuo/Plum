#include "fs_utils.hpp"
#include <sys/stat.h>
#include <filesystem>
#include <cstdlib>

bool ensure_dir(const std::string& path) {
	std::error_code ec;
	std::filesystem::create_directories(path, ec);
	return !ec;
}

bool file_exists(const std::string& path) {
	struct stat st{};
	return ::stat(path.c_str(), &st) == 0 && S_ISREG(st.st_mode);
}

bool unzip_zip(const std::string& zip_path, const std::string& out_dir) {
	// 简化：调用系统 unzip（依赖安装 unzip）
	std::string cmd = "unzip -o '" + zip_path + "' -d '" + out_dir + "' >/dev/null 2>&1";
	int rc = std::system(cmd.c_str());
	return rc == 0;
}


