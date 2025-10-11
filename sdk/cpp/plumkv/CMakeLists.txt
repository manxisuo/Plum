cmake_minimum_required(VERSION 3.16)
project(PlumKV)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Options
option(USE_GITHUB_MIRROR "Use GitHub mirror for dependencies" ON)  # 默认开启镜像

# GitHub mirror support
set(GITHUB_MIRROR "")
if(USE_GITHUB_MIRROR)
    set(GITHUB_MIRROR "https://mirror.ghproxy.com/")
    message(STATUS "Using GitHub mirror: ${GITHUB_MIRROR}")
endif()

# FetchContent for header-only libraries
include(FetchContent)

# nlohmann/json
FetchContent_Declare(
    json
    GIT_REPOSITORY ${GITHUB_MIRROR}https://github.com/nlohmann/json.git
    GIT_TAG v3.11.2
)
FetchContent_MakeAvailable(json)

# cpp-httplib
FetchContent_Declare(
    httplib
    GIT_REPOSITORY ${GITHUB_MIRROR}https://github.com/yhirose/cpp-httplib.git
    GIT_TAG v0.14.0
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

