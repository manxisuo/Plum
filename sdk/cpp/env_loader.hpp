#pragma once

#include <string>
#include <vector>
#include <fstream>
#include <sstream>
#include <cstdlib>
#include <unistd.h>

namespace plum {
namespace env {

/**
 * @brief 获取可执行程序所在目录
 */
static std::string getExeDir() {
    char path[1024];
    ssize_t len = readlink("/proc/self/exe", path, sizeof(path) - 1);
    if (len != -1) {
        path[len] = '\0';
        std::string exePath(path);
        size_t pos = exePath.find_last_of('/');
        if (pos != std::string::npos) {
            return exePath.substr(0, pos);
        }
    }
    return ".";
}

/**
 * @brief 从.env文件读取指定key的值
 * @return key对应的值，如果不存在返回空字符串
 */
static std::string readValue(const std::string& key, const std::string& envFile = "") {
    std::string file = envFile.empty() ? (getExeDir() + "/.env") : envFile;
    std::ifstream in(file);
    if (!in.is_open()) return "";
    
    std::string line;
    while (std::getline(in, line)) {
        size_t start = line.find_first_not_of(" \t");
        if (start == std::string::npos) continue;
        line = line.substr(start);
        if (line.empty() || line[0] == '#') continue;
        
        size_t pos = line.find('=');
        if (pos == std::string::npos) continue;
        
        std::string k = line.substr(0, pos);
        k.erase(k.find_last_not_of(" \t") + 1);
        
        if (k == key) {
            std::string value = line.substr(pos + 1);
            size_t vstart = value.find_first_not_of(" \t");
            if (vstart != std::string::npos) {
                value = value.substr(vstart);
                value.erase(value.find_last_not_of(" \t") + 1);
                
                // 去除引号
                if (value.size() >= 2) {
                    if ((value.front() == '"' && value.back() == '"') ||
                        (value.front() == '\'' && value.back() == '\'')) {
                        value = value.substr(1, value.size() - 2);
                    }
                }
            }
            return value;
        }
    }
    return "";
}

/**
 * @brief 检查.env文件中是否存在指定key
 */
static bool keyExists(const std::string& key, const std::string& envFile = "") {
    std::string file = envFile.empty() ? (getExeDir() + "/.env") : envFile;
    std::ifstream in(file);
    if (!in.is_open()) return false;
    
    std::string line;
    while (std::getline(in, line)) {
        size_t start = line.find_first_not_of(" \t");
        if (start == std::string::npos) continue;
        line = line.substr(start);
        if (line.empty() || line[0] == '#') continue;
        
        size_t pos = line.find('=');
        if (pos == std::string::npos) continue;
        
        std::string k = line.substr(0, pos);
        k.erase(k.find_last_not_of(" \t") + 1);
        
        if (k == key) return true;
    }
    return false;
}

/**
 * @brief 写入或更新key=value到.env文件
 */
static bool writeValue(const std::string& key, const std::string& value, const std::string& envFile = "") {
    std::string file = envFile.empty() ? (getExeDir() + "/.env") : envFile;
    
    // 读取现有内容
    std::vector<std::string> lines;
    bool keyFound = false;
    
    std::ifstream inFile(file);
    if (inFile.is_open()) {
        std::string line;
        while (std::getline(inFile, line)) {
            size_t start = line.find_first_not_of(" \t");
            if (start != std::string::npos) {
                std::string trimmed = line.substr(start);
                if (!trimmed.empty() && trimmed[0] != '#') {
                    size_t pos = trimmed.find('=');
                    if (pos != std::string::npos) {
                        std::string k = trimmed.substr(0, pos);
                        k.erase(k.find_last_not_of(" \t") + 1);
                        if (k == key) {
                            lines.push_back(key + "=" + value);
                            keyFound = true;
                            continue;
                        }
                    }
                }
            }
            lines.push_back(line);
        }
        inFile.close();
    }
    
    // key不存在，追加
    if (!keyFound) {
        if (!lines.empty() && !lines.back().empty()) {
            lines.push_back("");
        }
        lines.push_back("# Auto-generated");
        lines.push_back(key + "=" + value);
    }
    
    // 写回
    std::ofstream outFile(file);
    if (!outFile.is_open()) return false;
    
    for (const auto& line : lines) {
        outFile << line << "\n";
    }
    return true;
}

/**
 * @brief 简单的.env文件加载器
 * 
 * 支持格式：
 * - KEY=VALUE
 * - # 注释
 * - 空行
 * 
 * 优先级：环境变量 > .env文件 > 默认值
 * .env文件位置：可执行程序所在目录
 */
class EnvLoader {
public:
    /**
     * @brief 加载.env文件到进程环境变量
     * @param filePath .env文件路径（默认程序目录/.env）
     * @return 是否成功加载
     */
    static bool load(const std::string& filePath = "") {
        std::string envFile = filePath.empty() ? (getExeDir() + "/.env") : filePath;
        std::ifstream file(envFile);
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

