package widgets

import (
	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"
)

// Icon renders a registered icon.
func Icon(name string, size int, color string) El {
	return giode.Icon(name, St{Width: pr.Width(size), Height: pr.Height(size), Color: color})
}
