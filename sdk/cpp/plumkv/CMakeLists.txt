cmake_minimum_required(VERSION 3.16)
project(PlumKV)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Options
option(USE_GITHUB_MIRROR "Use GitHub mirror for dependencies" OFF)  # 默认关闭镜像

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

# Build plumkv library
add_library(plumkv STATIC
    DistributedMemory.cpp
)

target_include_directories(plumkv PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}
)

target_link_libraries(plumkv PUBLIC
    nlohmann_json::nlohmann_json
    httplib::httplib
    pthread
)

# Installation
install(TARGETS plumkv DESTINATION lib)
install(FILES DistributedMemory.hpp DESTINATION include/plumkv)

