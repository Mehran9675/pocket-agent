package widgets

import (
	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"
)

// Dot is a small status dot.
func Dot(color string, size int) El {
	return giode.Box(St{Width: pr.Width(size), Height: pr.Height(size), BorderRadius: pr.BorderRadius(size), Background: color})
}
