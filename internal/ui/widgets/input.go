package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// InputStyles use theme tokens so fields follow theme switches.
func InputStyles() St {
	return St{
		Background: "theme.surface", BorderColor: "theme.border", BorderWidth: 1,
		BorderRadius: 12, Padding: giode.SymmetricInset(14, 12), Color: "theme.text", FontSize: theme.SizeBody,
	}
}
