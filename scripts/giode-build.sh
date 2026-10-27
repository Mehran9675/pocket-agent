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
#
#   VERSION=1.0.0.3 scripts/giode-build.sh -o build/pocket-agent.apk android
set -euo pipefail
cd "$(dirname "$0")/.."

cp giode.toml giode.toml.orig
trap 'mv giode.toml.orig giode.toml' EXIT

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
open(path, "w").write("\n".join(lines))
PY

RELEASE=()
[ -n "${KEYSTORE:-}" ] && RELEASE=(-release)
${GIODE:-giode} build ${RELEASE[@]+"${RELEASE[@]}"} "$@"
