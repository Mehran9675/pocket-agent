package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// Card is a bordered surface.
func Card(st St, kids ...El) El {
	base := St{
		FlexDirection: giode.FlexDirectionColumn, Background: theme.T.Surface,
		BorderRadius: 14, BorderWidth: 1, BorderColor: theme.T.Border, Padding: giode.UniformInset(14), Gap: 10,
	}
	return giode.Box(base.With(st), kids...)
}
