#!/usr/bin/env bash
# Writes the engine manifest the app checks at launch, listing the
# engine builds in build/engine:
#
#   scripts/engine-manifest.sh https://github.com/OWNER/REPO/releases/download/TAG > engine-manifest.json
set -euo pipefail
source "$(dirname "$0")/llama-src.sh"

BASE=${1:?base URL}
ABI=$(sed -n 's/^#define PA_ABI_VERSION \([0-9]*\)/\1/p' "$ROOT/engine/pocketllama.h")
sum() { (sha256sum "$1" 2>/dev/null || shasum -a 256 "$1") | cut -d' ' -f1; }
size() { stat -c %s "$1" 2>/dev/null || stat -f %z "$1"; }

printf '{\n  "version": "%s",\n  "abi": %s,\n  "files": [' "$(engine_version)" "$ABI"
sep=""
for f in "$ROOT"/build/engine/libpocketllama-*; do
	name=$(basename "$f")
	IFS=- read -r _ os arch variant <<<"${name%.*}"
	printf '%s\n    {"os": "%s", "arch": "%s", "variant": "%s", "url": "%s/%s", "sha256": "%s", "size": %s}' \
		"$sep" "$os" "$arch" "$variant" "$BASE" "$name" "$(sum "$f")" "$(size "$f")"
	sep=","
done
printf '\n  ]\n}\n'
