cmake_minimum_required(VERSION 3.16)
project(PlumKVDemo)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# 添加plumkv SDK
add_subdirectory(${CMAKE_CURRENT_SOURCE_DIR}/../../sdk/cpp/plumkv ${CMAKE_CURRENT_BINARY_DIR}/plumkv)

# Demo应用
add_executable(kv-demo main.cpp)

# 设置include路径
target_include_directories(kv-demo PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../../sdk/cpp
)

target_link_libraries(kv-demo PRIVATE plumkv)

# 安装
install(TARGETS kv-demo DESTINATION .)

