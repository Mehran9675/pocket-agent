# Pocket Agent build targets. See docs/building.md.

APPID   ?= com.mehran9675.pocketagent
NAME    ?= Pocket Agent
VERSION ?= 1.0.0.1
ICON    := assets/appicon.png
OUT     := build
GOGIO   ?= gogio

# Set LLAMA=1 to include the on-device engine (run scripts/build-llama.sh first).
TAGS := $(if $(LLAMA),llama,)

.PHONY: run dev test vet desktop android ios ios-sim clean llama-android llama-ios

run:            ## run on the desktop
	go run -tags "$(TAGS)" .

dev:            ## run on the desktop with the offline Demo provider and the Giode MCP server
	GIODE_MCP=1 go run -tags "devmock $(TAGS)" .

test:
	go test ./...

vet:
	go vet ./...
	go vet -tags devmock ./...

desktop:
	go build -tags "$(TAGS)" -o $(OUT)/pocket-agent .

android:        ## APK for arm64 phones and tablets (Android 9+)
	$(GOGIO) -target android -arch arm64 -appid $(APPID) -name "$(NAME)" -version $(VERSION) \
		-icon $(ICON) -minsdk 28 -tags "$(TAGS)" $(if $(SIGNKEY),-signkey $(SIGNKEY) -signpass "$(SIGNPASS)",) \
		-o $(OUT)/pocket-agent.apk .

ios:            ## universal iPhone + iPad app; SIGNKEY = provisioning profile
	$(GOGIO) -target ios -arch arm64 -appid $(APPID) -name "$(NAME)" -version $(VERSION) \
		-icon $(ICON) -minsdk 15 -tags "$(TAGS)" $(if $(SIGNKEY),-signkey "$(SIGNKEY)",) \
		-o $(OUT)/PocketAgent.ipa .

ios-sim:        ## .app for the iOS simulator (without the on-device engine)
	$(GOGIO) -target ios -arch arm64 -appid $(APPID) -name "$(NAME)" -version $(VERSION) \
		-icon $(ICON) -minsdk 15 -o $(OUT)/PocketAgent.app .

llama-android:
	scripts/build-llama.sh android

llama-ios:
	scripts/build-llama.sh ios

clean:
	rm -rf $(OUT)
