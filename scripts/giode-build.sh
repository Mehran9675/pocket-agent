#!/usr/bin/env bash
# Runs `giode build` with the per-build values giode.toml can't take as
# flags, restoring giode.toml afterwards:
#
#   VERSION=1.2.0.7  app.version 1.2.0, app.build 7 (from a v1.2.0.7 tag)
#   TAGS=llama       Go build tags (iOS: compile the on-device engine in)
#   PROFILE=...      iOS provisioning profile
#   APPID=...        app ID, when signing needs a different one
#   KEYSTORE=...     Android release keystore (password in GIODE_KEYSTORE_PASS);
#                    builds with -release when set
#   GIO_PATCHES=false  build Android against plain Gio (60fps cap)
#
# Android builds use Gio with its frame pacing fixed (it otherwise caps
# 120Hz displays at 60fps): scripts/gio-patch.py makes a patched copy
# and go.mod points at it, with a replace, for the build. (The giode
# CLI's own patching passes -modfile in GOFLAGS, which breaks gogio's go
# commands that run outside the module.)
#
# In GitHub Actions a failed build's output is reported as an error
# annotation, and a failed Android build is retried once against plain
# Gio, so a release still gets an APK.
#
#   VERSION=1.0.0.3 scripts/giode-build.sh -o build/pocket-agent.apk android
set -euo pipefail
cd "$(dirname "$0")/.."

cp giode.toml giode.toml.orig
cp go.mod go.mod.orig
cp go.sum go.sum.orig
trap 'mv giode.toml.orig giode.toml; mv go.mod.orig go.mod; mv go.sum.orig go.sum' EXIT

configure() {
python3 - <<'PY'
import os, re
path = "giode.toml"
lines = open(path).read().split("\n")

def put(section, key, value):
    """Sets key in [section], adding the key or the section if missing."""
    start = next((i for i, l in enumerate(lines) if l.strip() == f"[{section}]"), None)
    if start is None:
        lines.extend(["", f"[{section}]", f"{key} = {value}"])
        return
    end = next((i for i in range(start + 1, len(lines)) if lines[i].startswith("[")), len(lines))
    for i in range(start + 1, end):
        if re.match(rf"\s*{re.escape(key)}\s*=", lines[i]):
            lines[i] = f"{key} = {value}"
            return
    lines.insert(start + 1, f"{key} = {value}")

q = lambda s: '"' + s.replace("\\", "\\\\").replace('"', '\\"') + '"'
v = os.environ.get("VERSION", "")
if re.fullmatch(r"\d+\.\d+\.\d+\.\d+", v):
    *ver, build = v.split(".")
    put("app", "version", q(".".join(ver)))
    put("app", "build", build)
if os.environ.get("APPID"):
    put("app", "id", q(os.environ["APPID"]))
if os.environ.get("TAGS"):
    put("app", "tags", q(os.environ["TAGS"]))
if os.environ.get("PROFILE"):
    put("ios", "profile", q(os.path.abspath(os.environ["PROFILE"])))
if os.environ.get("KEYSTORE"):
    put("android", "keystore", q(os.path.abspath(os.environ["KEYSTORE"])))
# The CLI's patching is replaced by scripts/gio-patch.py (see above).
put("android", "gio_patches", "false")
open(path, "w").write("\n".join(lines))
PY
}
configure

# patch_gio points go.mod at a patched copy of Gio for an Android build.
patch_gio() {
	local dir
	if dir=$(python3 scripts/gio-patch.py); then
		go mod edit -replace "gioui.org=$dir"
		echo "Building against Gio with Android frame pacing fixed ($dir)"
	else
		echo "::warning title=Android frame pacing::Gio couldn't be patched; this APK is capped at 60fps"
	fi
}
if [[ " $* " == *" android "* || " $* " == *" aab "* ]] && [ "${GIO_PATCHES:-}" != "false" ]; then
	patch_gio
fi

RELEASE=()
[ -n "${KEYSTORE:-}" ] && RELEASE=(-release)

LOG=$(mktemp)
build() {
	set +e
	${GIODE:-giode} build ${RELEASE[@]+"${RELEASE[@]}"} "$@" 2>&1 | tee "$LOG"
	local rc=${PIPESTATUS[0]}
	set -e
	return "$rc"
}

# report puts the end of a failed build's output in an annotation, which
# GitHub shows on the run's summary (and its API serves without the logs).
report() { # title
	[ -n "${GITHUB_ACTIONS:-}" ] || return 0
	local msg
	msg=$(tail -n 40 "$LOG" | sed 's/%/%25/g' | awk '{printf "%s%%0A", $0}')
	echo "::error title=$1::$msg"
}

if build "$@"; then
	exit 0
fi
report "giode build failed"
if [ -n "${GITHUB_ACTIONS:-}" ] && [[ " $* " == *" android "* ]] && [ "${GIO_PATCHES:-}" != "false" ]; then
	echo "::warning title=Android frame pacing::Retrying against plain Gio; this APK is capped at 60fps"
	cp giode.toml.orig giode.toml
	cp go.mod.orig go.mod
	cp go.sum.orig go.sum
	GIO_PATCHES=false configure
	build "$@" && exit 0
	report "giode build failed against plain Gio too"
fi
exit 1
