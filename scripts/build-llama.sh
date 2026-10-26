#!/usr/bin/env bash
# Builds llama.cpp as static libraries for Pocket Agent's on-device engine
# (internal/local, "llama" build tag).
#
#   scripts/build-llama.sh android   # arm64-v8a, needs ANDROID_NDK_ROOT (or ANDROID_HOME/ndk/<ver>)
#   scripts/build-llama.sh ios       # iPhone/iPad arm64 with Metal, needs Xcode
#   scripts/build-llama.sh darwin    # macOS (for desktop development), Metal
#   scripts/build-llama.sh linux     # Linux desktop
#
# Sources go to third_party/llama.cpp (pinned below), libraries to
# third_party/lib/<platform>, where internal/local/llama.go links them.
set -euo pipefail

LLAMA_REPO=${LLAMA_REPO:-https://github.com/ggml-org/llama.cpp}
# The commit the cgo binding was written against. Newer commits usually
# work; llama.h changes occasionally.
LLAMA_COMMIT=${LLAMA_COMMIT:-63b4b303bd87456adc2f09d56c7e1f74e9e26673}

ROOT=$(cd "$(dirname "$0")/.." && pwd)
SRC=$ROOT/third_party/llama.cpp
TARGET=${1:-}
[ -n "$TARGET" ] || { sed -n '2,12p' "$0"; exit 1; }

if [ ! -d "$SRC/.git" ]; then
	git clone --filter=blob:none "$LLAMA_REPO" "$SRC"
fi
git -C "$SRC" fetch --depth 1 origin "$LLAMA_COMMIT" 2>/dev/null || true
git -C "$SRC" checkout -q "$LLAMA_COMMIT"

COMMON=(
	-DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF
	-DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF -DLLAMA_BUILD_TOOLS=OFF
	-DLLAMA_BUILD_SERVER=OFF -DGGML_OPENMP=OFF -DGGML_NATIVE=OFF
)

build() { # build dir, lib dir, extra cmake args...
	local dir=$1 out=$2
	shift 2
	cmake -S "$SRC" -B "$dir" "${COMMON[@]}" "$@"
	cmake --build "$dir" --config Release -j "$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)" --target llama
	mkdir -p "$out"
	find "$dir" -name '*.a' -exec cp {} "$out/" \;
	echo "libraries in $out:"
	ls "$out"
}

case "$TARGET" in
android)
	NDK=${ANDROID_NDK_ROOT:-}
	if [ -z "$NDK" ] && [ -n "${ANDROID_HOME:-}" ]; then
		NDK=$(ls -d "$ANDROID_HOME"/ndk/* 2>/dev/null | sort -V | tail -1)
	fi
	[ -d "$NDK" ] || { echo "Set ANDROID_NDK_ROOT to your NDK"; exit 1; }
	OUT=$ROOT/third_party/lib/android_arm64
	build "$SRC/build-android-arm64" "$OUT" \
		-DCMAKE_TOOLCHAIN_FILE="$NDK/build/cmake/android.toolchain.cmake" \
		-DANDROID_ABI=arm64-v8a -DANDROID_PLATFORM=android-28 -DANDROID_STL=c++_static \
		-DGGML_CPU_ARM_ARCH=armv8.2-a+dotprod+fp16
	# cgo links the C++ runtime statically from the NDK.
	SYSROOT=$(ls -d "$NDK"/toolchains/llvm/prebuilt/*/sysroot | head -1)
	cp "$SYSROOT/usr/lib/aarch64-linux-android/libc++_static.a" "$SYSROOT/usr/lib/aarch64-linux-android/libc++abi.a" "$OUT/"
	;;
ios)
	build "$SRC/build-ios-arm64" "$ROOT/third_party/lib/ios_arm64" \
		-G Xcode -DCMAKE_SYSTEM_NAME=iOS -DCMAKE_OSX_ARCHITECTURES=arm64 \
		-DCMAKE_OSX_DEPLOYMENT_TARGET=15.0 -DCMAKE_XCODE_ATTRIBUTE_ONLY_ACTIVE_ARCH=NO \
		-DGGML_METAL=ON -DGGML_METAL_EMBED_LIBRARY=ON -DGGML_BLAS=ON
	;;
darwin)
	build "$SRC/build-darwin" "$ROOT/third_party/lib/darwin" \
		-DGGML_METAL=ON -DGGML_METAL_EMBED_LIBRARY=ON -DGGML_BLAS=ON
	;;
linux)
	build "$SRC/build-linux" "$ROOT/third_party/lib/linux"
	;;
*)
	echo "unknown target $TARGET"; exit 1 ;;
esac
