cmake_minimum_required(VERSION 3.16)

add_executable(grpc_echo_worker main.cpp)
target_link_libraries(grpc_echo_worker plumworker)
