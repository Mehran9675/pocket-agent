# Builds libpocketllama, the downloadable on-device engine: the C
# interface in pocketllama.h with llama.cpp linked in statically.
#
#   cmake -S engine -B build/engine -DLLAMA_SRC=third_party/llama.cpp
#   cmake --build build/engine --target pocketllama
#
# scripts/build-engine.sh wraps this for each platform.
cmake_minimum_required(VERSION 3.21)
project(pocketllama C CXX)

set(LLAMA_SRC "${CMAKE_CURRENT_SOURCE_DIR}/../third_party/llama.cpp" CACHE PATH "llama.cpp sources")
set(PA_ENGINE_VERSION "dev" CACHE STRING "engine version string")

set(BUILD_SHARED_LIBS OFF CACHE BOOL "" FORCE)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)
set(CMAKE_C_VISIBILITY_PRESET hidden)
set(CMAKE_CXX_VISIBILITY_PRESET hidden)
set(LLAMA_BUILD_TESTS OFF CACHE BOOL "" FORCE)
set(LLAMA_BUILD_EXAMPLES OFF CACHE BOOL "" FORCE)
set(LLAMA_BUILD_TOOLS OFF CACHE BOOL "" FORCE)
set(LLAMA_BUILD_SERVER OFF CACHE BOOL "" FORCE)
set(LLAMA_BUILD_COMMON OFF CACHE BOOL "" FORCE)
set(LLAMA_CURL OFF CACHE BOOL "" FORCE)
set(GGML_OPENMP OFF CACHE BOOL "" FORCE)
set(GGML_NATIVE OFF CACHE BOOL "" FORCE)
add_subdirectory(${LLAMA_SRC} llama.cpp EXCLUDE_FROM_ALL)

add_library(pocketllama SHARED pocketllama.c)
target_compile_definitions(pocketllama PRIVATE PA_ENGINE_VERSION="${PA_ENGINE_VERSION}")
target_link_libraries(pocketllama PRIVATE llama)
if(ANDROID)
	target_link_libraries(pocketllama PRIVATE log android)
	target_link_options(pocketllama PRIVATE -Wl,-z,max-page-size=16384 -Wl,--exclude-libs,ALL)
elseif(UNIX AND NOT APPLE)
	target_link_options(pocketllama PRIVATE -Wl,--exclude-libs,ALL)
endif()
