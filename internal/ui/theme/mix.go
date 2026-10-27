package theme

import "github.com/mehran9675/giode"

// Tint is color mixed into the surface color, opaque: soft badge and
// icon backgrounds. (Translucent fills look much stronger in Gio than
// on the web; see giode.Mix.)
func Tint(color string, t float64) string { return giode.Mix(color, T.Surface, t) }

// Mix blends fg over bg by t into an opaque color.
func Mix(fg, bg string, t float64) string { return giode.Mix(fg, bg, t) }
