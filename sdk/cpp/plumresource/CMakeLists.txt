cmake_minimum_required(VERSION 3.16)

add_library(plumresource STATIC
  src/plum_resource.cpp
)
target_include_directories(plumresource PUBLIC include)

# 支持离线模式（优先使用本地头文件）
option(USE_OFFLINE_DEPS "Use offline dependencies for C++ SDK" OFF)

if(USE_OFFLINE_DEPS)
    message(STATUS "使用离线模式，查找本地依赖...")
    message(STATUS "当前CMake目录: ${CMAKE_CURRENT_SOURCE_DIR}")
    
    # 检查是否有本地的nlohmann/json头文件
    find_path(NLOHMANN_JSON_INCLUDE_DIR
        NAMES nlohmann/json.hpp
        PATHS 
            ../third_party
            ../third_party/nlohmann
            ../../third_party
            ${CMAKE_CURRENT_SOURCE_DIR}/../third_party
            ${CMAKE_CURRENT_SOURCE_DIR}/../../third_party
            /usr/include/nlohmann
            /usr/local/include/nlohmann
        PATH_SUFFIXES nlohmann
    )
    
    message(STATUS "搜索结果: NLOHMANN_JSON_INCLUDE_DIR = ${NLOHMANN_JSON_INCLUDE_DIR}")
    
    if(NLOHMANN_JSON_INCLUDE_DIR)
        message(STATUS "找到本地nlohmann/json: ${NLOHMANN_JSON_INCLUDE_DIR}")
        target_include_directories(plumresource PUBLIC ${NLOHMANN_JSON_INCLUDE_DIR})
        
        # 在离线模式下，nlohmann_json目标应该已经在plumworker中创建
        # 这里只设置包含目录，不创建目标
        message(STATUS "使用已存在的nlohmann_json目标")
        
        # 查找cpp-httplib本地版本
        find_path(CPP_HTTPLIB_INCLUDE_DIR
            NAMES httplib.h
            PATHS 
                ../third_party/cpp-httplib
                ../third_party
                ${CMAKE_CURRENT_SOURCE_DIR}/../third_party/cpp-httplib
                ${CMAKE_CURRENT_SOURCE_DIR}/../third_party
                /usr/include/httplib
                /usr/local/include/httplib
        )
        
        if(CPP_HTTPLIB_INCLUDE_DIR)
            message(STATUS "找到本地cpp-httplib: ${CPP_HTTPLIB_INCLUDE_DIR}")
            target_include_directories(plumresource PUBLIC ${CPP_HTTPLIB_INCLUDE_DIR})
            
            # 在离线模式下，httplib目标应该已经在plumworker中创建
            # 这里只设置包含目录，不创建目标
            message(STATUS "使用已存在的httplib目标")
            
            # 链接httplib（在找到的情况下）
            target_link_libraries(plumresource PUBLIC httplib::httplib)
        else()
            message(WARNING "离线模式下未找到cpp-httplib，可能需要手动提供")
        endif()
        
        # 链接nlohmann_json
        target_link_libraries(plumresource PUBLIC nlohmann_json::nlohmann_json)
    else()
        message(STATUS "搜索路径调试:")
        message(STATUS "  检查 ../third_party/nlohmann: ${CMAKE_CURRENT_SOURCE_DIR}/../third_party/nlohmann")
        message(STATUS "  检查 ../../third_party/nlohmann: ${CMAKE_CURRENT_SOURCE_DIR}/../../third_party/nlohmann")
        if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/../third_party")
            message(STATUS "  ../third_party 目录存在")
        else()
            message(STATUS "  ../third_party 目录不存在")
        endif()
        if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/../../third_party")
            message(STATUS "  ../../third_party 目录存在")
        else()
            message(STATUS "  ../../third_party 目录不存在")
        endif()
        message(FATAL_ERROR "离线模式下未找到nlohmann/json.hpp，请提供本地副本")
    endif()
else()
    # 在线模式：使用FetchContent下载依赖
    include(FetchContent)
    set(FETCHCONTENT_UPDATES_DISCONNECTED ON)

    FetchContent_Declare(
      json
      GIT_REPOSITORY https://github.com/nlohmann/json.git
      GIT_TAG v3.11.3
    )
    FetchContent_MakeAvailable(json)
    target_link_libraries(plumresource PUBLIC nlohmann_json::nlohmann_json)

    FetchContent_Declare(
      httplib
      GIT_REPOSITORY https://github.com/yhirose/cpp-httplib.git
      GIT_TAG v0.15.3
    )
    FetchContent_MakeAvailable(httplib)
    target_link_libraries(plumresource PUBLIC httplib::httplib)
endif()

# 链接pthread库（cpp-httplib需要）
target_link_libraries(plumresource PUBLIC pthread)
