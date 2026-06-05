cmake_minimum_required(VERSION 3.20)

project(KhaApp VERSION 0.1.0 LANGUAGES CXX)

add_subdirectory(app)

add_custom_target(daemon_build
    COMMAND go build -o ${CMAKE_BINARY_DIR}/khaapp-daemon ./cmd
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/daemon
    COMMENT "Building khaapp-daemon"
)
