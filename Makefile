# Pocket Agent build targets. See docs/building.md.

# App name, ID, icon, permissions and Info.plist texts are in giode.toml; packaging uses the
# giode CLI (go install github.com/mehran9675/giode-cli/cmd/giode@latest).
VERSION ?= 1.0.0.1
OUT     := build
export GIODE ?= giode

# iOS only: set LLAMA=1 to compile the on-device engine into the app (run
# scripts/build-llama.sh ios first). Android and desktops download it.
TAGS := $(strip $(if $(LLAMA),llama,) $(GOTAGS))

# GOTAGS adds Go build tags, e.g. GOTAGS=noopengl to make Gio render with Vulkan on Android.

# A locally built engine for desktop runs (make engine).
HOST_ENGINE = $(firstword $(wildcard $(OUT)/engine/libpocketllama-linux-*.so $(OUT)/engine/libpocketllama-darwin-*.dylib))

.PHONY: run dev test vet desktop android ios ios-sim clean engine engine-android llama-ios

run:            ## run on the desktop
	POCKET_AGENT_ENGINE=$(HOST_ENGINE) go run .

dev:            ## run on the desktop with the offline Demo provider and the Giode MCP server
	POCKET_AGENT_ENGINE=$(HOST_ENGINE) GIODE_MCP=1 go run -tags devmock .

test:
	go test ./...

vet:
	go vet ./...
	go vet -tags devmock ./...

desktop:
	mkdir -p $(OUT)
	go build -o $(OUT)/pocket-agent .

android:        ## APK for arm64 phones and tablets (Android 9+); SIGNKEY/SIGNPASS for a release keystore
	mkdir -p $(OUT)
	VERSION=$(VERSION) TAGS="$(TAGS)" KEYSTORE=$(SIGNKEY) GIODE_KEYSTORE_PASS="$(SIGNPASS)" \
		scripts/giode-build.sh -o $(OUT)/pocket-agent.apk android

ios:            ## universal iPhone + iPad app; SIGNKEY = provisioning profile
	mkdir -p $(OUT)
	VERSION=$(VERSION) TAGS=$(TAGS) PROFILE=$(SIGNKEY) APPID=$(APPID) \
		scripts/giode-build.sh -o $(OUT)/PocketAgent.ipa ios

ios-sim:        ## .app for the iOS simulator (without the on-device engine)
	mkdir -p $(OUT)
	VERSION=$(VERSION) scripts/giode-build.sh -o $(OUT)/PocketAgent.app ios-sim

engine:         ## the on-device engine for this computer
	scripts/build-engine.sh $(if $(filter Darwin,$(shell uname -s)),darwin,linux)

engine-android: ## the on-device engine builds for Android phones
	scripts/build-engine.sh android armv8.2
	scripts/build-engine.sh android armv8

llama-ios:
	scripts/build-llama.sh ios

clean:
	rm -rf $(OUT)
