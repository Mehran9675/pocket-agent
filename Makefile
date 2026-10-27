# Pocket Agent build targets. See docs/building.md.

APPID   ?= com.mehran9675.pocketagent
NAME    ?= Pocket Agent
VERSION ?= 1.0.0.1
ICON    := assets/appicon.png
OUT     := build
GOGIO   ?= gogio

# iOS only: set LLAMA=1 to compile the on-device engine into the app (run
# scripts/build-llama.sh ios first). Android and desktops download it.
TAGS := $(if $(LLAMA),llama,)

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

android:        ## APK for arm64 phones and tablets (Android 9+)
	mkdir -p $(OUT)
	$(GOGIO) -target android -arch arm64 -appid $(APPID) -name "$(NAME)" -version $(VERSION) \
		-icon $(ICON) -minsdk 28 $(if $(SIGNKEY),-signkey $(SIGNKEY) -signpass "$(SIGNPASS)",) \
		-o $(OUT)/pocket-agent.apk .

ios:            ## universal iPhone + iPad app; SIGNKEY = provisioning profile
	mkdir -p $(OUT)
	$(GOGIO) -target ios -arch arm64 -appid $(APPID) -name "$(NAME)" -version $(VERSION) \
		-icon $(ICON) -minsdk 15 -tags "$(TAGS)" $(if $(SIGNKEY),-signkey "$(SIGNKEY)",) \
		-o $(OUT)/PocketAgent.ipa .
	scripts/ios-plist.sh $(OUT)/PocketAgent.ipa

ios-sim:        ## .app for the iOS simulator (without the on-device engine)
	mkdir -p $(OUT)
	# gogio builds simulator apps for Intel only; they run on Apple silicon Macs too.
	$(GOGIO) -target ios -arch amd64 -appid $(APPID) -name "$(NAME)" -version $(VERSION) \
		-icon $(ICON) -minsdk 15 -o $(OUT)/PocketAgent.app .
	scripts/ios-plist.sh $(OUT)/PocketAgent.app

engine:         ## the on-device engine for this computer
	scripts/build-engine.sh $(if $(filter Darwin,$(shell uname -s)),darwin,linux)

engine-android: ## the on-device engine builds for Android phones
	scripts/build-engine.sh android armv8.2
	scripts/build-engine.sh android armv8

llama-ios:
	scripts/build-llama.sh ios

clean:
	rm -rf $(OUT)
