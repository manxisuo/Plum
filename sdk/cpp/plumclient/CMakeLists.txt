cmake_minimum_required(VERSION 3.16)
project(plumclient)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# 查找依赖库
find_package(PkgConfig REQUIRED)
find_package(Threads REQUIRED)

# 支持离线模式（优先使用本地头文件）
option(USE_OFFLINE_DEPS "Use offline dependencies for C++ SDK" OFF)

if(USE_OFFLINE_DEPS)
    message(STATUS "使用离线模式，查找本地依赖...")
    message(STATUS "当前CMake目录: ${CMAKE_CURRENT_SOURCE_DIR}")
    
    # 检查是否有本地的nlohmann/json头文件
    find_path(NLOHMANN_JSON_INCLUDE_DIR
        NAMES nlohmann/json.hpp
        PATHS 
            ../third_party/nlohmann
            ../../third_party/nlohmann
            ${CMAKE_CURRENT_SOURCE_DIR}/../third_party/nlohmann
            ${CMAKE_CURRENT_SOURCE_DIR}/../../third_party/nlohmann
            /usr/include/nlohmann
            /usr/local/include/nlohmann
    )
    
    message(STATUS "搜索结果: NLOHMANN_JSON_INCLUDE_DIR = ${NLOHMANN_JSON_INCLUDE_DIR}")
    
    if(NLOHMANN_JSON_INCLUDE_DIR)
        message(STATUS "找到本地nlohmann/json: ${NLOHMANN_JSON_INCLUDE_DIR}")
        set(PLUMCLIENT_NLOHMANN_JSON_INCLUDE_DIR ${NLOHMANN_JSON_INCLUDE_DIR})
        
        # 在离线模式下，nlohmann_json目标应该已经在plumworker中创建
        # 这里只设置包含目录，不创建目标
        message(STATUS "使用已存在的nlohmann_json目标")
        
        # 查找cpp-httplib本地版本
        find_path(CPP_HTTPLIB_INCLUDE_DIR
            NAMES httplib.h
            PATHS 
                ../third_party/cpp-httplib
                ../../third_party/cpp-httplib
                ${CMAKE_CURRENT_SOURCE_DIR}/../third_party/cpp-httplib
                ${CMAKE_CURRENT_SOURCE_DIR}/../../third_party/cpp-httplib
                /usr/include/httplib
                /usr/local/include/httplib
        )
        
        if(CPP_HTTPLIB_INCLUDE_DIR)
            message(STATUS "找到本地cpp-httplib: ${CPP_HTTPLIB_INCLUDE_DIR}")
            set(PLUMCLIENT_HTTPLIB_INCLUDE_DIR ${CPP_HTTPLIB_INCLUDE_DIR})
            
            # 在离线模式下，httplib目标应该已经在plumworker中创建
            # 这里只设置包含目录，不创建目标
            message(STATUS "使用已存在的httplib目标")
        else()
            message(WARNING "离线模式下未找到cpp-httplib，可能需要手动提供")
        endif()
    else()
        message(FATAL_ERROR "离线模式下未找到nlohmann/json.hpp，请提供本地副本")
    endif()
else()
    # 在线模式：使用FetchContent下载依赖
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
        message(WARNING "httplib not found, trying to use from build directory")
        set(HTTPLIB_INCLUDE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../../build/_deps/httplib-src")
    endif()

    # 确保路径正确
    if(NOT EXISTS "${HTTPLIB_INCLUDE_DIR}/httplib.h")
        message(FATAL_ERROR "httplib.h not found at ${HTTPLIB_INCLUDE_DIR}")
    endif()
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
        ${NLOHMANN_JSON_INCLUDE_DIR}
        ${HTTPLIB_INCLUDE_DIR}
)

# 在离线模式下设置额外的包含目录
if(USE_OFFLINE_DEPS)
    if(PLUMCLIENT_NLOHMANN_JSON_INCLUDE_DIR)
        target_include_directories(plumclient PUBLIC ${PLUMCLIENT_NLOHMANN_JSON_INCLUDE_DIR})
    endif()
    if(PLUMCLIENT_HTTPLIB_INCLUDE_DIR)
        target_include_directories(plumclient PUBLIC ${PLUMCLIENT_HTTPLIB_INCLUDE_DIR})
    endif()
endif()

# 链接库
target_link_libraries(plumclient
    PUBLIC
        Threads::Threads
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
