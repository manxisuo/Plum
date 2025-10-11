#pragma once

#include <string>
#include <map>
#include <memory>
#include <functional>
#include <mutex>
#include <thread>
#include <atomic>
#include <vector>

namespace plum {
namespace kv {

/**
 * @brief 分布式KV存储客户端
 * 
 * Plum的分布式KV存储提供集群级别的键值对存储能力，
 * 结合了持久化的可靠性和内存缓存的快速访问特性。
 * 
 * 核心特性：
 * - 持久化存储：数据保存在Controller的SQLite中，不会丢失
 * - 快速访问：本地缓存提供内存般的读取速度
 * - 命名空间隔离：多应用/实例互不干扰
 * - 实时同步：SSE推送保证多节点数据一致性
 * - 类型安全：支持 string/int/double/bool
 * - 崩溃恢复：支持应用崩溃后状态恢复
 * 
 * 使用示例：
 * @code
 * auto kv = DistributedMemory::create("my-app");
 * kv->putInt("counter", 100);
 * int count = kv->getInt("counter", 0);
 * @endcode
 */
class DistributedMemory {
public:
    /**
     * @brief 工厂方法：创建分布式KV存储实例
     * @param ns 命名空间（建议使用 appName 或 instanceId）
     * @param controllerURL Controller地址（默认从环境变量读取）
     * @return DistributedMemory实例（分布式KV存储客户端）
     */
    static std::shared_ptr<DistributedMemory> create(
        const std::string& ns,
        const std::string& controllerURL = ""
    );

    ~DistributedMemory();

    // ===== 通用接口 =====
    
    /**
     * @brief 存储键值对
     * @param key 键
     * @param value 值
     * @return 是否成功
     */
    bool put(const std::string& key, const std::string& value);
    
    /**
     * @brief 获取键值
     * @param key 键
     * @param defaultValue 默认值（key不存在时返回）
     * @return 值
     */
    std::string get(const std::string& key, const std::string& defaultValue = "");
    
    /**
     * @brief 检查键是否存在
     */
    bool exists(const std::string& key);
    
    /**
     * @brief 删除键
     */
    bool remove(const std::string& key);
    
    // ===== 类型化接口 =====
    
    bool putInt(const std::string& key, int64_t value);
    int64_t getInt(const std::string& key, int64_t defaultValue = 0);
    
    bool putDouble(const std::string& key, double value);
    double getDouble(const std::string& key, double defaultValue = 0.0);
    
    bool putBool(const std::string& key, bool value);
    bool getBool(const std::string& key, bool defaultValue = false);
    
    // ===== 二进制数据 =====
    
    /**
     * @brief 存储二进制数据（使用Base64编码）
     * @param key 键
     * @param data 二进制数据指针
     * @param size 数据大小（字节）
     * @return 是否成功
     */
    bool putBytes(const std::string& key, const void* data, size_t size);
    
    /**
     * @brief 存储二进制数据（vector版本）
     */
    bool putBytes(const std::string& key, const std::vector<uint8_t>& data);
    
    /**
     * @brief 获取二进制数据
     * @param key 键
     * @param defaultValue 默认值
     * @return 二进制数据
     */
    std::vector<uint8_t> getBytes(const std::string& key, const std::vector<uint8_t>& defaultValue = {});
    
    /**
     * @brief 获取二进制数据（C风格接口）
     * @param key 键
     * @param buffer 输出缓冲区
     * @param size 输入：缓冲区大小；输出：实际数据大小
     * @return 是否成功（false表示key不存在或buffer太小）
     */
    bool getBytes(const std::string& key, void* buffer, size_t& size);
    
    // ===== 批量操作 =====
    
    /**
     * @brief 获取所有键值对
     */
    std::map<std::string, std::string> getAll();
    
    /**
     * @brief 批量存储
     */
    bool putBatch(const std::map<std::string, std::string>& kvs);
    
    /**
     * @brief 刷新缓存（从Controller重新加载所有数据）
     */
    void refresh();
    
    /**
     * @brief 订阅变更回调
     * @param callback 当有键值变更时调用
     */
    void subscribe(std::function<void(const std::string& key, const std::string& value)> callback);
    
    /**
     * @brief 获取命名空间
     */
    std::string getNamespace() const { return namespace_; }

private:
    DistributedMemory(const std::string& ns, const std::string& controllerURL);
    
    void preloadCache();
    void startSSE();
    void stopSSE();
    void onSSEEvent(const std::string& event, const std::string& data);
    
    std::string buildURL(const std::string& path) const;
    bool httpPut(const std::string& key, const std::string& value, const std::string& type);
    std::string httpGet(const std::string& key, bool& found);
    bool httpDelete(const std::string& key);
    
    std::string namespace_;
    std::string controllerURL_;
    
    // 本地缓存
    std::map<std::string, std::string> cache_;
    std::map<std::string, std::string> types_; // 记录类型
    mutable std::mutex cacheMutex_;
    
    // SSE相关
    std::thread sseThread_;
    std::atomic<bool> sseRunning_;
    
    // 变更回调
    std::vector<std::function<void(const std::string&, const std::string&)>> callbacks_;
    std::mutex callbackMutex_;
};

} // namespace kv
} // namespace plum

