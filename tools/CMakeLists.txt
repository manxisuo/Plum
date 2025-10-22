cmake_minimum_required(VERSION 3.16)
project(cpp_weak_network_test LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# 查找依赖
find_package(PkgConfig REQUIRED)
pkg_check_modules(CURL REQUIRED libcurl)
pkg_check_modules(JSONCPP REQUIRED jsoncpp)

# 包含目录
include_directories(${CURL_INCLUDE_DIRS})
include_directories(${JSONCPP_INCLUDE_DIRS})

# 添加可执行文件
add_executable(cpp_weak_network_test cpp_weak_network_test.cpp)

# 链接库
target_link_libraries(cpp_weak_network_test 
    ${CURL_LIBRARIES}
    ${JSONCPP_LIBRARIES}
    pthread
)

# 编译选项
target_compile_options(cpp_weak_network_test PRIVATE ${CURL_CFLAGS_OTHER})
target_compile_options(cpp_weak_network_test PRIVATE ${JSONCPP_CFLAGS_OTHER})

# 设置输出目录
set_target_properties(cpp_weak_network_test PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}
)
