cmake_minimum_required(VERSION 3.10)
project(PlumDemoApp)

set(CMAKE_CXX_STANDARD 11)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_executable(demo-app main.cpp)

# 安装目标
install(TARGETS demo-app DESTINATION .)

