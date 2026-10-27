# Building Pocket Agent

Pocket Agent is a Go program built with Gio's `gogio` tool, which packages it as an Android
APK or an iOS/iPadOS app. One iOS build runs on both iPhone and iPad (gogio declares device
families 1 and 2), adapting through the app's responsive layouts.

## Prerequisites

| | |
| --- | --- |
| Go | 1.26.6 or later |
| gogio | `go install gioui.org/cmd/gogio@latest` |
| Android | Android SDK (platform 35, build-tools), NDK r27+, JDK 17+. Set `ANDROID_HOME` (and `ANDROID_NDK_ROOT` if the NDK lives elsewhere). |
| iOS / iPadOS | macOS with Xcode 15+, an Apple developer account and a provisioning profile for the app ID. |
| Desktop (development) | Linux: Wayland/X11/EGL/Vulkan headers ([Gio install guide](https://gioui.org/doc/install)). macOS and Windows need nothing extra. |
| On-device engine | CMake 3.21+ and a C/C++ toolchain (the NDK for Android, Xcode for iOS). Only needed to build the engine yourself. |

## Desktop development

```sh
make run            # the real app
make dev            # with the offline Demo provider and Giode's MCP server on :7600
make test vet
```

`POCKET_AGENT_DATA=/tmp/pa` points the app at a scratch data directory. `make engine` builds the
on-device engine for this computer into `build/engine/`; `make run` and `make dev` use it. With `make dev`, an
MCP client (Claude Code, for example) can inspect, click, type into and screenshot the
running app - the screenshots in this repository were taken that way at phone (400×860 dp)
and iPad (1366×1024 dp) sizes.

## Android

```sh
make android
adb install -r build/pocket-agent.apk
```

- Targets arm64 devices running Android 9 (API 28) or later; set `-arch` in the Makefile to add
  `amd64` for emulators.
- Release signing: `make android SIGNKEY=release.jks SIGNPASS=...`. Without a key, gogio signs
  with the debug keystore. Bump the last component of `VERSION` (the version code) for every
  store upload.

## iOS and iPadOS

```sh
make ios SIGNKEY=/path/to/PocketAgent.mobileprovision
```

- `APPID` must match the provisioning profile (`make ios APPID=com.example.pocketagent ...`).
- `make ios-sim` builds a simulator `.app` (without the on-device engine).
- Install the `.ipa` with Xcode (Devices and Simulators), Apple Configurator or `xcrun devicectl`.

## On-device engine (llama.cpp)

The engine is not part of the Android app. It is `engine/`: a small C interface over llama.cpp
(`pocketllama.h`, versioned by `PA_ABI_VERSION`), built as a shared library with llama.cpp
linked in. The app downloads it on demand and keeps it current:

1. At every launch `local.Runtime` fetches `engine-manifest.json` from the latest GitHub
   release (`POCKET_AGENT_ENGINE_MANIFEST` overrides the URL).
2. It picks the build for the device: on Android `armv8.2` (dot product and fp16, phones from
   about 2018 on) when `/proc/cpuinfo` reports `asimddp` and `asimdhp`, otherwise `armv8`.
3. The first install waits for the user (Library → *Download engine*); later versions install
   by themselves. Downloads are checked against the manifest's SHA-256 and size, stored
   read-only under `<data>/engine/<version>/`, and switched in by rewriting `current.json`.
   A version already loaded in memory keeps running until the next launch.
4. The app `dlopen`s the library when a model is first loaded, and refuses one whose ABI
   differs from its own.

```sh
make engine-android     # build/engine/libpocketllama-android-arm64-{armv8.2,armv8}.so
make engine             # this computer (Linux or macOS)
scripts/engine-manifest.sh https://example.com/files > engine-manifest.json
```

`scripts/llama-src.sh` pins the llama.cpp commit (`LLAMA_COMMIT`). An engine's version is that
commit plus a hash of `engine/`, so apps update exactly when the engine changes. The release
workflow builds every variant, writes the manifest and attaches them all to the release. Note
that Google Play doesn't allow apps to download native code; this suits builds distributed
through GitHub releases.

**iOS** can't load downloaded code, so the engine is compiled into the app:

```sh
make llama-ios && make ios LLAMA=1 SIGNKEY=...
```

`scripts/build-llama.sh ios` builds llama.cpp's static libraries (Metal, shader library
embedded) and the `llama` tag compiles `engine/pocketllama.c` into the app
(`internal/local/bridge_static.c`). Without the tag the iOS app runs cloud and network models
only.

Models are GGUF files. The catalog in `internal/local/catalog.go` lists Q4_K_M quantizations
of 0.6-4B parameter models; the library also searches Hugging Face and imports `.gguf` files
from the device. Split GGUFs and vision projectors aren't supported yet.

To test an engine build against a model:

```sh
PA_TEST_ENGINE=build/engine/libpocketllama-linux-amd64.so PA_TEST_MODEL=model.gguf go test ./internal/local -run Native
```

## App icon and name

`assets/appicon.png` (1024×1024, full-bleed - the OS applies its own mask) is passed to gogio
with `-icon`. The display name comes from `NAME` in the Makefile.

## Troubleshooting

- **"engine isn't installed"** in the model library: download it there. If the download
  fails, check that the latest release has `engine-manifest.json` and a build for the device.
- **iOS link errors for `ggml_*`** symbols: rebuild the libraries (`scripts/build-llama.sh
  ios`); the library order is fixed in `internal/local/builtin_ios.go`.
- **Models download but fail to load**: the model is larger than the device can hold with the
  chosen context size - lower it in Settings → On-device engine, or pick a smaller model.
- **Web search returns nothing**: DuckDuckGo rate-limits bursts; switch to Brave, Tavily or a
  SearXNG instance in Settings → Web search.
