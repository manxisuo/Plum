cmake_minimum_required(VERSION 3.16)
project(plumclient)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# 查找依赖库
find_package(PkgConfig REQUIRED)
find_package(Threads REQUIRED)

# 查找libcurl
pkg_check_modules(CURL REQUIRED libcurl)

# 查找nlohmann/json (已通过主CMakeLists.txt处理)

# 查找nlohmann/json
find_path(NLOHMANN_JSON_INCLUDE_DIR
    NAMES nlohmann/json.hpp
    PATHS 
        /usr/include 
        /usr/local/include
        ${CMAKE_CURRENT_SOURCE_DIR}/../../build/_deps/json-src/include
    PATH_SUFFIXES nlohmann
)

if(NOT NLOHMANN_JSON_INCLUDE_DIR)
    message(WARNING "nlohmann/json not found, trying to use from build directory")
    set(NLOHMANN_JSON_INCLUDE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../../build/_deps/json-src/include")
endif()

# 创建库
add_library(plumclient SHARED
    src/plum_client.cpp
    src/service_client.cpp
    src/discovery_client.cpp
    src/weak_network_support.cpp
    src/cache.cpp
)

# 设置包含目录
target_include_directories(plumclient
    PUBLIC
        include
        ${CURL_INCLUDE_DIRS}
        ${NLOHMANN_JSON_INCLUDE_DIR}
)

# 链接库
target_link_libraries(plumclient
    PUBLIC
        ${CURL_LIBRARIES}
        Threads::Threads
)

# 设置编译选项
target_compile_options(plumclient
    PRIVATE
        ${CURL_CFLAGS_OTHER}
)

# 安装规则
install(TARGETS plumclient
    LIBRARY DESTINATION lib
    ARCHIVE DESTINATION lib
)

install(DIRECTORY include/
    DESTINATION include/plumclient
    FILES_MATCHING PATTERN "*.hpp"
)

# 示例程序在 examples/ 目录中单独构建
