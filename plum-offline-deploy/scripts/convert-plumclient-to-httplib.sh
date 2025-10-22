#!/bin/bash
# 将plumclient从libcurl转换为httplib的脚本
# 用于简化依赖管理

echo "🔧 将plumclient从libcurl转换为httplib..."

# 检查当前plumclient实现
if [ ! -f "../source/Plum/sdk/cpp/plumclient/src/plum_client.cpp" ]; then
    echo "❌ 未找到plumclient源码"
    exit 1
fi

echo "📋 当前plumclient使用libcurl，需要转换为httplib"
echo ""
echo "需要修改的文件："
echo "1. sdk/cpp/plumclient/src/plum_client.cpp"
echo "2. sdk/cpp/plumclient/src/service_client.cpp" 
echo "3. sdk/cpp/plumclient/src/discovery_client.cpp"
echo "4. sdk/cpp/plumclient/CMakeLists.txt"
echo "5. sdk/cpp/plumclient/include/plum_client.hpp"
echo ""
echo "主要修改内容："
echo "- 替换 #include <curl/curl.h> 为 #include <httplib.h>"
echo "- 替换 CURL* 为 httplib::Client"
echo "- 替换 curl_easy_* 函数为 httplib::Client 方法"
echo "- 更新CMake配置，移除libcurl依赖"
echo ""
echo "优势："
echo "✅ 统一使用httplib，与plum_resource一致"
echo "✅ 减少系统依赖，不需要libcurl开发包"
echo "✅ 简化部署和构建"
echo "✅ header-only库，更容易管理"
echo ""
echo "风险："
echo "⚠️  需要重写HTTP请求代码"
echo "⚠️  需要测试功能完整性"
echo "⚠️  可能需要调整错误处理逻辑"
echo ""
echo "建议："
echo "1. 先备份当前实现"
echo "2. 逐步替换HTTP请求代码"
echo "3. 保持API接口不变"
echo "4. 充分测试所有功能"
