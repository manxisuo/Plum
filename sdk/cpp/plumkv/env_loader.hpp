#pragma once

#include <string>
#include <map>
#include <fstream>
#include <sstream>
#include <cstdlib>

namespace plum {
namespace env {

/**
 * @brief 简单的.env文件加载器
 * 
 * 支持格式：
 * - KEY=VALUE
 * - # 注释
 * - 空行
 * 
 * 优先级：环境变量 > .env文件 > 默认值
 */
class EnvLoader {
public:
    /**
     * @brief 加载.env文件到进程环境变量
     * @param filePath .env文件路径（默认当前目录）
     * @return 是否成功加载
     */
    static bool load(const std::string& filePath = ".env") {
        std::ifstream file(filePath);
        if (!file.is_open()) {
            return false; // 文件不存在不算错误
        }
        
        std::string line;
        int count = 0;
        while (std::getline(file, line)) {
            // 去除首尾空格
            line = trim(line);
            
            // 跳过空行和注释
            if (line.empty() || line[0] == '#') continue;
            
            // 解析 KEY=VALUE
            size_t pos = line.find('=');
            if (pos == std::string::npos) continue;
            
            std::string key = trim(line.substr(0, pos));
            std::string value = trim(line.substr(pos + 1));
            
            // 去除引号（支持 KEY="VALUE" 格式）
            if (value.size() >= 2) {
                if ((value.front() == '"' && value.back() == '"') ||
                    (value.front() == '\'' && value.back() == '\'')) {
                    value = value.substr(1, value.size() - 2);
                }
            }
            
            // 只有环境变量未设置时才从.env加载
            if (!key.empty() && std::getenv(key.c_str()) == nullptr) {
                setenv(key.c_str(), value.c_str(), 0); // 0 = 不覆盖已存在的
                count++;
            }
        }
        
        return count > 0;
    }
    
private:
    static std::string trim(const std::string& str) {
        size_t start = 0;
        size_t end = str.size();
        
        while (start < end && std::isspace(str[start])) start++;
        while (end > start && std::isspace(str[end - 1])) end--;
        
        return str.substr(start, end - start);
    }
};

} // namespace env
} // namespace plum

