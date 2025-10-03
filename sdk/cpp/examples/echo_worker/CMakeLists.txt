cmake_minimum_required(VERSION 3.16)

add_executable(echo_worker main.cpp)
target_link_libraries(echo_worker PRIVATE plumworker)
target_compile_definitions(echo_worker PRIVATE CONTROLLER_BASE_DEFAULT="http://127.0.0.1:8080")


