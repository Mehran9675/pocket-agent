#!/usr/bin/env bash
# Builds llama.cpp as static libraries for iOS and iPadOS, where apps
# can't load downloaded code and the engine is compiled into the app
# (internal/local/bridge_static.c, "llama" build tag). Android and
# desktops download the engine instead; see scripts/build-engine.sh.
#
#   scripts/build-llama.sh ios   # iPhone/iPad arm64 with Metal, needs Xcode
#
# Libraries go to third_party/lib/ios_arm64, where
# internal/local/builtin_ios.go links them.
set -euo pipefail
source "$(dirname "$0")/llama-src.sh"

[ "${1:-}" = ios ] || { sed -n '2,11p' "$0"; exit 1; }
fetch_llama

DIR=$SRC/build-ios-arm64
OUT=$ROOT/third_party/lib/ios_arm64
cmake -S "$SRC" -B "$DIR" -G Xcode \
	-DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF \
	-DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF -DLLAMA_BUILD_TOOLS=OFF \
	-DLLAMA_BUILD_SERVER=OFF -DLLAMA_BUILD_COMMON=OFF -DLLAMA_CURL=OFF \
	-DGGML_OPENMP=OFF -DGGML_NATIVE=OFF \
	-DCMAKE_SYSTEM_NAME=iOS -DCMAKE_OSX_ARCHITECTURES=arm64 \
	-DCMAKE_OSX_DEPLOYMENT_TARGET=15.0 -DCMAKE_XCODE_ATTRIBUTE_ONLY_ACTIVE_ARCH=NO \
	-DGGML_METAL=ON -DGGML_METAL_EMBED_LIBRARY=ON -DGGML_BLAS=ON
cmake --build "$DIR" --config Release -j "$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)" --target llama
mkdir -p "$OUT"
find "$DIR" -name '*.a' -exec cp {} "$OUT/" \;
echo "libraries in $OUT:"
ls "$OUT"
