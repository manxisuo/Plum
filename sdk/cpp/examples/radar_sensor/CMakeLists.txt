cmake_minimum_required(VERSION 3.16)

add_executable(radar_sensor main.cpp)
target_link_libraries(radar_sensor PRIVATE plumresource pthread)
target_compile_definitions(radar_sensor PRIVATE CONTROLLER_BASE_DEFAULT="http://127.0.0.1:8080")
