cmake_minimum_required(VERSION 3.16)
project(plum_agent LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

add_executable(plum_agent
    src/main.cpp
    src/http_client.hpp
    src/http_client.cpp
    src/reconciler.hpp
    src/reconciler.cpp
    src/fs_utils.hpp
    src/fs_utils.cpp
)

target_include_directories(plum_agent PRIVATE src)

find_package(CURL REQUIRED)
target_link_libraries(plum_agent PRIVATE CURL::libcurl)


