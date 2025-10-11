cmake_minimum_required(VERSION 3.16)
project(PlumWorkerDemo)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# 查找gRPC和Protobuf
find_package(PkgConfig REQUIRED)
pkg_check_modules(GRPC REQUIRED grpc++)
pkg_check_modules(PROTOBUF REQUIRED protobuf)

# 添加proto生成代码
set(PROTO_PATH "${CMAKE_CURRENT_SOURCE_DIR}/../../sdk/cpp/grpc/proto")
add_library(task_proto STATIC
    ${PROTO_PATH}/task_service.pb.cc
    ${PROTO_PATH}/task_service.grpc.pb.cc
)
target_include_directories(task_proto PUBLIC 
    ${CMAKE_CURRENT_SOURCE_DIR}/../../sdk/cpp/grpc
    ${GRPC_INCLUDE_DIRS}
    ${PROTOBUF_INCLUDE_DIRS}
)
target_link_libraries(task_proto PUBLIC ${GRPC_LIBRARIES} ${PROTOBUF_LIBRARIES})

# Demo应用
add_executable(worker-demo main.cpp)
target_include_directories(worker-demo PRIVATE 
    ${CMAKE_CURRENT_SOURCE_DIR}/../../sdk/cpp/grpc
)
target_link_libraries(worker-demo PRIVATE task_proto)

# 安装
install(TARGETS worker-demo DESTINATION .)

