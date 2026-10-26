package theme

import (
	"fmt"

	pr "github.com/mehran9675/giode/styles/properties"
)

// Mix blends fg over bg by t (0..1) in sRGB space and returns an opaque
// color. Gio composites translucent colors in linear light, which makes
// faint overlays (a 5% white hover) look several times stronger than
// on the web, so the theme precomputes tints as opaque colors instead.
func Mix(fg, bg string, t float64) string {
	a, b := pr.ResolveColor(fg), pr.ResolveColor(bg)
	ch := func(x, y uint8) uint8 { return uint8(float64(y) + (float64(x)-float64(y))*t + 0.5) }
	return fmt.Sprintf("#%02x%02x%02x", ch(a.R, b.R), ch(a.G, b.G), ch(a.B, b.B))
}

// Tint is color mixed into the surface color: soft badge and icon
// backgrounds.
func Tint(color string, t float64) string { return Mix(color, T.Surface, t) }
