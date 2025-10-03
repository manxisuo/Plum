cmake_minimum_required(VERSION 3.16)
project(plumworker-cpp LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_subdirectory(plumworker)
add_subdirectory(examples/echo_worker)


