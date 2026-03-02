cmake_minimum_required(VERSION 3.20)
project(shelternav C)

set(CMAKE_C_STANDARD 17)
set(CMAKE_C_STANDARD_REQUIRED ON)
set(CMAKE_C_EXTENSIONS OFF)

# ---------------------------------------------------------------------------
# Compiler flags
# ---------------------------------------------------------------------------
add_compile_options(-Wall -Wextra -Werror -O3 -flto)

# ---------------------------------------------------------------------------
# ASan option for debug builds
# ---------------------------------------------------------------------------
option(SN_ENABLE_ASAN "Enable AddressSanitizer in debug builds" OFF)

if(SN_ENABLE_ASAN AND CMAKE_BUILD_TYPE STREQUAL "Debug")
    add_compile_options(-fsanitize=address -fno-omit-frame-pointer)
    add_link_options(-fsanitize=address)
endif()

# ---------------------------------------------------------------------------
# Platform / architecture detection for ASM files
# ---------------------------------------------------------------------------
set(SN_ASM_SOURCES "")

if(CMAKE_SYSTEM_PROCESSOR MATCHES "x86_64|AMD64")
    message(STATUS "Detected x86_64 — enabling AVX2 / x86 ASM stubs")
    enable_language(ASM_NASM)
    list(APPEND SN_ASM_SOURCES
        asm/haversine_avx2.asm
        asm/heap_x86.asm
    )
elseif(CMAKE_SYSTEM_PROCESSOR MATCHES "aarch64|arm64|ARM64")
    message(STATUS "Detected ARM64 — enabling NEON / arm64 ASM stubs")
    enable_language(ASM)
    list(APPEND SN_ASM_SOURCES
        asm/haversine_neon.S
        asm/heap_arm64.S
    )
else()
    message(WARNING "Unknown architecture: ${CMAKE_SYSTEM_PROCESSOR} — no ASM files included")
endif()

# ---------------------------------------------------------------------------
# Collect C sources
# ---------------------------------------------------------------------------
file(GLOB SN_C_SOURCES CONFIGURE_DEPENDS src/*.c)

# ---------------------------------------------------------------------------
# Static library target
# ---------------------------------------------------------------------------
add_library(shelternav STATIC ${SN_C_SOURCES} ${SN_ASM_SOURCES})

target_include_directories(shelternav
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include>
)

# Link math library (required for sin/cos/sqrt/atan2)
target_link_libraries(shelternav PUBLIC m)

# ---------------------------------------------------------------------------
# Install rules
# ---------------------------------------------------------------------------
install(TARGETS shelternav
    ARCHIVE DESTINATION lib
)
install(DIRECTORY include/ DESTINATION include)

# ---------------------------------------------------------------------------
# Testing (ctest)
# ---------------------------------------------------------------------------
enable_testing()

# Placeholder: tests will be added under tests/ and registered here.
# Example:
#   add_executable(test_kdtree tests/test_kdtree.c)
#   target_link_libraries(test_kdtree PRIVATE shelternav)
#   add_test(NAME test_kdtree COMMAND test_kdtree)
