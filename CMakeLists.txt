cmake_minimum_required(VERSION 3.25)
project(vulkango VERSION 0.0.1 LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 26)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS ON)

set(CMAKE_C_STANDARD 23)
set(CMAKE_C_STANDARD_REQUIRED ON)

find_package(Vulkan QUIET)
find_package(Threads REQUIRED)

include(FetchContent)

#FetchContent_Declare(
#    SDL3
#    GIT_REPOSITORY https://github.com/libsdl-org/SDL.git
#    GIT_TAG "release-3.2.26"
#    GIT_SHALLOW TRUE
#)
#FetchContent_MakeAvailable(SDL3)

FetchContent_Declare(
    volk
    GIT_REPOSITORY https://github.com/zeux/volk.git
    GIT_SHALLOW TRUE
)

# Configure volk with position independent code before making it available
set(CMAKE_POSITION_INDEPENDENT_CODE ON)
FetchContent_MakeAvailable(volk)


# Set position independent code for all targets when building shared library
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# Build shared library for CGO bindings
add_library(vulkango SHARED
    vulkango.cpp
)

target_compile_definitions(vulkango PUBLIC
    VK_NO_PROTOTYPES
)

target_link_libraries(vulkango PUBLIC
    volk
)
