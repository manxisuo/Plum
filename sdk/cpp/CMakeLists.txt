cmake_minimum_required(VERSION 3.16)
project(plum-cpp LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_subdirectory(plumworker)
add_subdirectory(plumresource)
add_subdirectory(plumkv)
add_subdirectory(plumclient)
add_subdirectory(examples/echo_worker)
add_subdirectory(examples/radar_sensor)
add_subdirectory(examples/service_client_example)


