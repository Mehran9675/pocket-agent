#!/usr/bin/env bash
# Builds the downloadable on-device engine: libpocketllama (engine/), the
# C interface the app loads at run time, with llama.cpp linked in.
#
#   scripts/build-engine.sh android armv8.2   # arm64 phones with dot product (about 2018 on)
#   scripts/build-engine.sh android armv8     # any arm64 phone
#   scripts/build-engine.sh linux             # this Linux machine's architecture
#   scripts/build-engine.sh darwin            # macOS, Metal
#
# Android needs ANDROID_NDK_ROOT (or ANDROID_HOME/ndk/<version>). Files go
# to build/engine/libpocketllama-<os>-<arch>[-<variant>].<so|dylib>;
# scripts/engine-manifest.sh lists them for the app's updater.
set -euo pipefail
source "$(dirname "$0")/llama-src.sh"

TARGET=${1:-}
VARIANT=${2:-}
[ -n "$TARGET" ] || { sed -n '2,13p' "$0"; exit 1; }
fetch_llama
VERSION=$(engine_version)
OUT=$ROOT/build/engine
mkdir -p "$OUT"
JOBS=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)

build() { # build dir, output name, extra cmake args...
	local dir=$1 name=$2
	shift 2
	cmake -S "$ROOT/engine" -B "$dir" -DCMAKE_BUILD_TYPE=Release -DLLAMA_SRC="$SRC" -DPA_ENGINE_VERSION="$VERSION" "$@"
	cmake --build "$dir" --config Release -j "$JOBS" --target pocketllama
	local lib
	lib=$(find "$dir" -maxdepth 2 -name 'libpocketllama.so' -o -maxdepth 2 -name 'libpocketllama.dylib' | head -1)
	cp "$lib" "$OUT/$name"
	echo "engine $VERSION: $OUT/$name"
}

case "$TARGET" in
android)
	NDK=${ANDROID_NDK_ROOT:-}
	if [ -z "$NDK" ] && [ -n "${ANDROID_HOME:-}" ]; then
		NDK=$(ls -d "$ANDROID_HOME"/ndk/* 2>/dev/null | sort -V | tail -1)
	fi
	[ -d "$NDK" ] || { echo "Set ANDROID_NDK_ROOT to your NDK"; exit 1; }
	case "$VARIANT" in
	armv8.2) ARCH=armv8.2-a+dotprod+fp16 ;;
	armv8) ARCH=armv8-a ;;
	*) echo "android variant: armv8.2 or armv8"; exit 1 ;;
	esac
	build "$ROOT/build/engine-android-$VARIANT" "libpocketllama-android-arm64-$VARIANT.so" \
		-DCMAKE_TOOLCHAIN_FILE="$NDK/build/cmake/android.toolchain.cmake" \
		-DANDROID_ABI=arm64-v8a -DANDROID_PLATFORM=android-28 -DANDROID_STL=c++_static \
		-DGGML_CPU_ARM_ARCH="$ARCH"
	"$NDK"/toolchains/llvm/prebuilt/*/bin/llvm-strip "$OUT/libpocketllama-android-arm64-$VARIANT.so"
	;;
linux)
	case "$(uname -m)" in x86_64) A=amd64 ;; aarch64) A=arm64 ;; *) A=$(uname -m) ;; esac
	build "$ROOT/build/engine-linux" "libpocketllama-linux-$A.so"
	strip "$OUT/libpocketllama-linux-$A.so"
	;;
darwin)
	build "$ROOT/build/engine-darwin" "libpocketllama-darwin-arm64.dylib" \
		-DCMAKE_OSX_ARCHITECTURES=arm64 -DGGML_METAL=ON -DGGML_METAL_EMBED_LIBRARY=ON -DGGML_BLAS=ON
	;;
*)
	echo "unknown target $TARGET"; exit 1 ;;
esac
