cmake_minimum_required(VERSION 3.10)
project(libengine)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(OpenSSL REQUIRED)

add_library(engine SHARED engine.cpp)
target_include_directories(engine PUBLIC ${CMAKE_CURRENT_SOURCE_DIR} ${OPENSSL_INCLUDE_DIR})
target_link_libraries(engine PRIVATE OpenSSL::Crypto)
set_target_properties(engine PROPERTIES OUTPUT_NAME "engine")
