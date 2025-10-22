cmake_minimum_required(VERSION 3.16)
project(PlumKV)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Options
option(USE_GITHUB_MIRROR "Use GitHub mirror for dependencies" OFF)  # 默认关闭镜像
option(USE_OFFLINE_DEPS "Use offline dependencies for C++ SDK" OFF)

# 支持离线模式（优先使用本地头文件）
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
        set(PLUMKV_NLOHMANN_JSON_INCLUDE_DIR ${NLOHMANN_JSON_INCLUDE_DIR})
        
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
            set(PLUMKV_HTTPLIB_INCLUDE_DIR ${CPP_HTTPLIB_INCLUDE_DIR})
            
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
    # GitHub mirror support
    set(GITHUB_MIRROR "")
    if(USE_GITHUB_MIRROR)
        set(GITHUB_MIRROR "https://ghproxy.link/")
        message(STATUS "Using GitHub mirror: ${GITHUB_MIRROR}")
    else()
        message(STATUS "Downloading from GitHub directly (use -DUSE_GITHUB_MIRROR=ON to enable mirror)")
    endif()

    # FetchContent for header-only libraries
    include(FetchContent)

    # nlohmann/json
    FetchContent_Declare(
        json
        URL https://github.com/nlohmann/json/archive/refs/tags/v3.11.2.tar.gz
    )
    FetchContent_MakeAvailable(json)

    # cpp-httplib
    FetchContent_Declare(
        httplib
        URL https://github.com/yhirose/cpp-httplib/archive/refs/tags/v0.14.0.tar.gz
    )
    FetchContent_MakeAvailable(httplib)
endif()

# Build plumkv library
add_library(plumkv STATIC
    DistributedMemory.cpp
)

target_include_directories(plumkv PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}
)

# 在离线模式下设置额外的包含目录
if(USE_OFFLINE_DEPS)
    if(PLUMKV_NLOHMANN_JSON_INCLUDE_DIR)
        target_include_directories(plumkv PUBLIC ${PLUMKV_NLOHMANN_JSON_INCLUDE_DIR})
    endif()
    if(PLUMKV_HTTPLIB_INCLUDE_DIR)
        target_include_directories(plumkv PUBLIC ${PLUMKV_HTTPLIB_INCLUDE_DIR})
    endif()
endif()

target_link_libraries(plumkv PUBLIC
    nlohmann_json::nlohmann_json
    httplib::httplib
    pthread
)

# Installation
install(TARGETS plumkv DESTINATION lib)
install(FILES DistributedMemory.hpp DESTINATION include/plumkv)

