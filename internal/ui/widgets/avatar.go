package widgets

import (
	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"
)

// Avatar shows a name's initials on a stable color, size dp across.
func Avatar(name string, size int) El {
	return giode.Avatar(name, St{Width: pr.Width(size)})
}
