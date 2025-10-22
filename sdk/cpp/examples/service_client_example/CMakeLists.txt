cmake_minimum_required(VERSION 3.16)
project(service_client_example)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# 查找依赖库
find_package(PkgConfig REQUIRED)
find_package(Threads REQUIRED)

# 使用httplib (header-only库，无需查找)

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
    set(NLOHMANN_JSON_INCLUDE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../../build/_deps/json-src/include")
endif()

# 查找httplib
find_path(HTTPLIB_INCLUDE_DIR
    NAMES httplib.h
    PATHS 
        /usr/include 
        /usr/local/include
        ${CMAKE_CURRENT_SOURCE_DIR}/../../build/_deps/httplib-src
        ${CMAKE_CURRENT_SOURCE_DIR}/../../third_party/cpp-httplib
)

if(NOT HTTPLIB_INCLUDE_DIR)
    set(HTTPLIB_INCLUDE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../../build/_deps/httplib-src")
endif()

# 创建示例程序
add_executable(service_client_example
    main.cpp
    ../../plumclient/src/plum_client.cpp
    ../../plumclient/src/service_client.cpp
    ../../plumclient/src/discovery_client.cpp
    ../../plumclient/src/weak_network_support.cpp
    ../../plumclient/src/cache.cpp
)

# 设置包含目录
target_include_directories(service_client_example
    PRIVATE
        ../../plumclient/include
        ${NLOHMANN_JSON_INCLUDE_DIR}
        ${HTTPLIB_INCLUDE_DIR}
)

# 链接库
target_link_libraries(service_client_example
    PRIVATE
        Threads::Threads
)
