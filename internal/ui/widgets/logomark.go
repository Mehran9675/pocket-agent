package widgets

import (
	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// LogoMark is the app's mark: a gradient tile with a spark.
func LogoMark(size int) El {
	return giode.Box(St{
		Width: pr.Width(size), Height: pr.Height(size), BorderRadius: pr.BorderRadius(size * 3 / 10),
		BackgroundGradient: giode.Linear(135, theme.T.Primary, theme.Mix(theme.T.Primary, theme.T.Bg, 0.7)),
		Align:              giode.AlignCenter, Justify: giode.JustifyCenter,
	}, Icon("sparkles", size*11/20, theme.T.OnPrimary))
}
