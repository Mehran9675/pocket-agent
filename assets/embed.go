// Package assets embeds Pocket Agent's fonts and icons.
package assets

import "embed"

// Fonts holds Inter (UI) and JetBrains Mono (code), both under the SIL
// Open Font License.
//
//go:embed fonts/*.ttf
var Fonts embed.FS

// Icons holds the Lucide icon subset the app uses (ISC license).
//
//go:embed icons/*.svg
var Icons embed.FS
