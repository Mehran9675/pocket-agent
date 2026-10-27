#!/usr/bin/env python3
"""Makes a copy of the Gio module this project uses with Android frame
pacing fixed, and prints its directory.

Gio asks Android's Choreographer for the next frame only after
presenting the current one; presenting blocks until a vsync, so the
request misses the next one and a 120Hz display gets 60 frames a
second. The patch asks before drawing (the same change the giode CLI
makes). Exits non-zero, with the reason on stderr, when Gio can't be
patched.
"""
import json
import os
import shutil
import stat
import subprocess
import sys
import tempfile

REVISION = "1"

PATCHES = [
    (
        "app/os_android.go",
        """func (w *window) draw(env *C.JNIEnv, sync bool) {
	if !w.visible {
		return
	}
""",
        """func (w *window) draw(env *C.JNIEnv, sync bool) {
	if !w.visible {
		return
	}
	// pocket-agent: ask for the next frame before drawing this one, so
	// the request makes the next vsync.
	if !sync && !w.animating {
		return
	}
	wasAnimating := w.animating
	if wasAnimating {
		callVoidMethod(env, w.view, gioView.postFrameCallback)
	}
""",
    ),
    (
        "app/os_android.go",
        """		Sync: sync,
	})
	if w.animating {
		callVoidMethod(env, w.view, gioView.postFrameCallback)
	}
""",
        """		Sync: sync,
	})
	if w.animating && !wasAnimating {
		callVoidMethod(env, w.view, gioView.postFrameCallback)
	}
""",
    ),
]


def go(*args):
    return subprocess.run(["go", *args], check=True, capture_output=True, text=True).stdout


def fail(msg):
    print(msg, file=sys.stderr)
    sys.exit(1)


def main():
    mod = json.loads(go("list", "-m", "-json", "gioui.org"))
    if mod.get("Replace"):
        fail("go.mod already replaces gioui.org")
    if not mod.get("Dir"):
        go("mod", "download", "gioui.org")
        mod = json.loads(go("list", "-m", "-json", "gioui.org"))
    src = mod.get("Dir")
    if not src:
        fail("can't find Gio's source")

    base = os.environ.get("RUNNER_TEMP") or tempfile.gettempdir()
    dst = os.path.join(base, f"gio-{mod['Version']}-pocketagent{REVISION}")
    if os.path.exists(os.path.join(dst, "go.mod")):
        print(dst)
        return
    tmp = tempfile.mkdtemp(dir=base, prefix="gio-")
    try:
        shutil.copytree(src, tmp, dirs_exist_ok=True)
        for root, dirs, files in os.walk(tmp):  # the module cache is read-only
            for n in dirs + files:
                p = os.path.join(root, n)
                os.chmod(p, os.stat(p).st_mode | stat.S_IWUSR)
        for rel, old, new in PATCHES:
            path = os.path.join(tmp, rel)
            text = open(path).read()
            if text.count(old) != 1:
                fail(f"Gio {mod['Version']}: {rel} doesn't have the code the patch expects")
            open(path, "w").write(text.replace(old, new, 1))
        os.rename(tmp, dst)
    finally:
        shutil.rmtree(tmp, ignore_errors=True)
    print(dst)


if __name__ == "__main__":
    main()
