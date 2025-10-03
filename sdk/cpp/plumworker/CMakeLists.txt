cmake_minimum_required(VERSION 3.16)

add_library(plumworker STATIC
  src/plum_worker.cpp
)
target_include_directories(plumworker PUBLIC include)

# FetchContent for nlohmann/json and cpp-httplib (header-only)
include(FetchContent)
set(FETCHCONTENT_UPDATES_DISCONNECTED ON)

FetchContent_Declare(
  json
  GIT_REPOSITORY https://github.com/nlohmann/json.git
  GIT_TAG v3.11.3
)
FetchContent_MakeAvailable(json)
target_link_libraries(plumworker PUBLIC nlohmann_json::nlohmann_json)

FetchContent_Declare(
  httplib
  GIT_REPOSITORY https://github.com/yhirose/cpp-httplib.git
  GIT_TAG v0.15.3
)
FetchContent_MakeAvailable(httplib)
target_link_libraries(plumworker PUBLIC httplib::httplib)


