cmake_minimum_required(VERSION 3.16)
project(PlumWorkerDemo)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# 添加 plumworker SDK
add_subdirectory(../../sdk/cpp/plumworker plumworker)

# Demo应用（使用 SDK）
add_executable(worker-demo main.cpp)
target_link_libraries(worker-demo PRIVATE plumworker)

# 安装
install(TARGETS worker-demo DESTINATION .)

