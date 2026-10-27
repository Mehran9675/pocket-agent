# Sourced by the engine scripts: fetch_llama checks out llama.cpp at the
# pinned commit into third_party/llama.cpp; engine_version names a build.

LLAMA_REPO=${LLAMA_REPO:-https://github.com/ggml-org/llama.cpp}
# The commit engine/pocketllama.c was written against. Newer commits
# usually work; llama.h changes occasionally.
LLAMA_COMMIT=${LLAMA_COMMIT:-63b4b303bd87456adc2f09d56c7e1f74e9e26673}

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SRC=$ROOT/third_party/llama.cpp

fetch_llama() {
	if [ ! -d "$SRC/.git" ]; then
		git clone --filter=blob:none "$LLAMA_REPO" "$SRC"
	fi
	if [ "$(git -C "$SRC" rev-parse HEAD)" != "$LLAMA_COMMIT" ]; then
		git -C "$SRC" fetch --depth 1 origin "$LLAMA_COMMIT" 2>/dev/null || git -C "$SRC" fetch origin
		git -C "$SRC" checkout -q "$LLAMA_COMMIT"
	fi
}

# engine_version names an engine build: the llama.cpp commit and a hash
# of the engine sources. Apps install any version that differs from
# theirs, so it changes exactly when the engine does.
engine_version() {
	local h
	h=$(cat "$ROOT"/engine/pocketllama.c "$ROOT"/engine/pocketllama.h "$ROOT"/engine/CMakeLists.txt | (sha256sum 2>/dev/null || shasum -a 256) | cut -c1-8)
	echo "${LLAMA_COMMIT:0:8}.$h"
}
