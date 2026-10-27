package widgets

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/segmented"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// Segmented is a segmented control kept under key, filling its row.
// icons, when given, name an icon per option.
func Segmented(key string, options []string, icons []string, selected int, onChange func(int)) El {
	c := giode.Keyed("segmented", key+"|"+strings.Join(options, "|"), func() *segmented.Control {
		return giode.Input.Segmented(options...)
	})
	sel := theme.T.Elevated
	if theme.T.Dark {
		sel = theme.T.BorderStrong
	}
	c.Styles(St{Width: -1, Background: theme.T.Surface2, BorderColor: sel, Color: theme.T.Text, FontSize: theme.SizeFootnote})
	if len(icons) > 0 {
		els := make([]El, len(icons))
		for i, name := range icons {
			els[i] = Icon(name, 15, theme.T.Text2)
		}
		c.Icons(els...)
	}
	if c.Selected() != selected {
		c.SetSelected(selected)
	}
	c.OnChange(onChange)
	return c
}
