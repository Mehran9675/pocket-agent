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
| On-device engine | CMake 3.21+ and a C/C++ toolchain (the NDK for Android, Xcode for iOS). |

## Desktop development

```sh
make run            # the real app
make dev            # with the offline Demo provider and Giode's MCP server on :7600
make test vet
```

`POCKET_AGENT_DATA=/tmp/pa` points the app at a scratch data directory. With `make dev`, an
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

The engine is compiled in only with the `llama` build tag, so the default builds need no C++
toolchain; without it the model library explains that the engine isn't included and suggests
Ollama / LM Studio.

```sh
make llama-android && make android LLAMA=1
make llama-ios     && make ios LLAMA=1 SIGNKEY=...
scripts/build-llama.sh darwin && make run LLAMA=1    # macOS desktop, Metal
scripts/build-llama.sh linux  && make run LLAMA=1    # Linux desktop, CPU
```

`scripts/build-llama.sh` clones llama.cpp at the commit the binding was written against
(`LLAMA_COMMIT` in the script) into `third_party/llama.cpp`, builds static libraries and copies
them to `third_party/lib/<platform>`, where the cgo directives in `internal/local/llama.go` look
for them. Android builds target ARMv8.2 with dot-product instructions (all phones from about
2018 on); iOS builds use Metal with the shader library embedded.

Models are GGUF files. The catalog in `internal/local/catalog.go` lists Q4_K_M quantizations
of 0.6-4B parameter models; the library also searches Hugging Face and imports `.gguf` files
from the device. Split GGUFs and vision projectors aren't supported yet.

## App icon and name

`assets/appicon.png` (1024×1024, full-bleed - the OS applies its own mask) is passed to gogio
with `-icon`. The display name comes from `NAME` in the Makefile.

## Troubleshooting

- **"no on-device engine"** in the model library: the build lacks `-tags llama`, or the
  libraries weren't built for that platform.
- **Link errors for `ggml_*`** symbols: rebuild the libraries for the target
  (`scripts/build-llama.sh <platform>`); the library order is fixed in `llama.go`.
- **Models download but fail to load**: the model is larger than the device can hold with the
  chosen context size - lower it in Settings → On-device engine, or pick a smaller model.
- **Web search returns nothing**: DuckDuckGo rate-limits bursts; switch to Brave, Tavily or a
  SearXNG instance in Settings → Web search.
