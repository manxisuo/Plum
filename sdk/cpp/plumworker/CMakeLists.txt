cmake_minimum_required(VERSION 3.16)

# Find gRPC and protobuf packages
find_package(PkgConfig REQUIRED)
pkg_check_modules(GRPC REQUIRED grpc++)
pkg_check_modules(PROTOBUF REQUIRED protobuf)

add_library(plumworker STATIC
  src/plum_worker.cpp
  src/grpc_worker.cpp
)
target_include_directories(plumworker PUBLIC include)
target_include_directories(plumworker PUBLIC ../grpc/proto)

# Add gRPC generated files
add_library(grpc_proto STATIC
  ../grpc/proto/task_service.pb.cc
  ../grpc/proto/task_service.grpc.pb.cc
)
target_include_directories(grpc_proto PUBLIC ../grpc)
target_include_directories(grpc_proto PUBLIC ${GRPC_INCLUDE_DIRS})
target_include_directories(grpc_proto PUBLIC ${PROTOBUF_INCLUDE_DIRS})

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
        target_include_directories(plumworker PUBLIC ${NLOHMANN_JSON_INCLUDE_DIR})
        
        # 创建nlohmann_json库的别名（离线模式）
        if(NOT TARGET nlohmann_json)
            add_library(nlohmann_json INTERFACE)
            target_include_directories(nlohmann_json INTERFACE ${NLOHMANN_JSON_INCLUDE_DIR})
            add_library(nlohmann_json::nlohmann_json ALIAS nlohmann_json)
            set(NLOHMANN_JSON_TARGET_CREATED TRUE CACHE INTERNAL "nlohmann_json target created")
        endif()
        
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
            target_include_directories(plumworker PUBLIC ${CPP_HTTPLIB_INCLUDE_DIR})
            
            # 创建httplib库的别名（离线模式）
            if(NOT TARGET httplib)
                add_library(httplib INTERFACE)
                target_include_directories(httplib INTERFACE ${CPP_HTTPLIB_INCLUDE_DIR})
                add_library(httplib::httplib ALIAS httplib)
                set(HTTPLIB_TARGET_CREATED TRUE CACHE INTERNAL "httplib target created")
            endif()
            
            # 链接httplib（在找到的情况下）
            target_link_libraries(plumworker PUBLIC httplib::httplib)
        else()
            message(WARNING "离线模式下未找到cpp-httplib，可能需要手动提供")
        endif()
        
        # 链接nlohmann_json
        target_link_libraries(plumworker PUBLIC nlohmann_json::nlohmann_json)
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

    # 支持使用GitHub镜像（中国网络环境）
    option(USE_GITHUB_MIRROR "Use GitHub mirror for China network" OFF)
    if(USE_GITHUB_MIRROR)
        set(GITHUB_MIRROR "https://ghproxy.link/")
    else()
        set(GITHUB_MIRROR "")
    endif()

    FetchContent_Declare(
        json
        GIT_REPOSITORY ${GITHUB_MIRROR}https://github.com/nlohmann/json.git
        GIT_TAG v3.11.3
    )
    FetchContent_MakeAvailable(json)
    target_link_libraries(plumworker PUBLIC nlohmann_json::nlohmann_json)

    FetchContent_Declare(
        httplib
        GIT_REPOSITORY ${GITHUB_MIRROR}https://github.com/yhirose/cpp-httplib.git
        GIT_TAG v0.15.3
    )
    FetchContent_MakeAvailable(httplib)
    target_link_libraries(plumworker PUBLIC httplib::httplib)
endif()

# Link gRPC and protobuf libraries
target_link_libraries(plumworker PUBLIC grpc_proto)
target_link_libraries(grpc_proto PUBLIC ${GRPC_LIBRARIES} ${PROTOBUF_LIBRARIES})
target_compile_options(grpc_proto PUBLIC ${GRPC_CFLAGS_OTHER} ${PROTOBUF_CFLAGS_OTHER})


