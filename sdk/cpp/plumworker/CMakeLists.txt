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

# FetchContent for nlohmann/json and cpp-httplib (header-only)
include(FetchContent)
set(FETCHCONTENT_UPDATES_DISCONNECTED ON)

# 支持使用GitHub镜像（中国网络环境）
# 使用方法: cmake -DUSE_GITHUB_MIRROR=ON ..
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

# Link gRPC and protobuf libraries
target_link_libraries(plumworker PUBLIC grpc_proto)
target_link_libraries(grpc_proto PUBLIC ${GRPC_LIBRARIES} ${PROTOBUF_LIBRARIES})
target_compile_options(grpc_proto PUBLIC ${GRPC_CFLAGS_OTHER} ${PROTOBUF_CFLAGS_OTHER})


