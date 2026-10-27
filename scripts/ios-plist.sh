#!/usr/bin/env bash
# Adds the Info.plist usage descriptions iOS requires before an app may
# use the camera, microphone and speech recognition (photo attachments
# and dictation), then re-signs the app. gogio writes Info.plist itself
# and can't add them.
#
#   scripts/ios-plist.sh build/PocketAgent.app   # simulator build, ad-hoc signed
#   scripts/ios-plist.sh build/PocketAgent.ipa   # device build, signed with SIGN_IDENTITY
#                                                # (default: the first code signing identity)
set -euo pipefail

TARGET=${1:?path to a .app or .ipa}

add_keys() { # Info.plist
	plutil -replace NSCameraUsageDescription -string "Pocket Agent uses the camera to attach photos you take to a message." "$1"
	plutil -replace NSMicrophoneUsageDescription -string "Pocket Agent uses the microphone so you can dictate messages." "$1"
	plutil -replace NSSpeechRecognitionUsageDescription -string "Pocket Agent turns your speech into text so you can dictate messages." "$1"
}

case "$TARGET" in
*.app)
	add_keys "$TARGET/Info.plist"
	codesign -f -s - "$TARGET"
	;;
*.ipa)
	IDENTITY=${SIGN_IDENTITY:-$(security find-identity -v -p codesigning | awk 'NR==1 {print $2}')}
	[ -n "$IDENTITY" ] || { echo "no code signing identity; set SIGN_IDENTITY"; exit 1; }
	WORK=$(mktemp -d)
	trap 'rm -rf "$WORK"' EXIT
	unzip -q "$TARGET" -d "$WORK"
	APP=$(ls -d "$WORK"/Payload/*.app | head -1)
	# Keep the entitlements gogio signed with.
	codesign -d --entitlements - --xml "$APP" > "$WORK/entitlements.plist" 2>/dev/null || true
	add_keys "$APP/Info.plist"
	if [ -s "$WORK/entitlements.plist" ]; then
		codesign -f -s "$IDENTITY" --entitlements "$WORK/entitlements.plist" "$APP"
	else
		codesign -f -s "$IDENTITY" "$APP"
	fi
	OUT=$(cd "$(dirname "$TARGET")" && pwd)/$(basename "$TARGET")
	rm -f "$OUT"
	(cd "$WORK" && zip -qry "$OUT" Payload)
	;;
*)
	echo "expected a .app or .ipa"; exit 1 ;;
esac
echo "added usage descriptions to $TARGET"
