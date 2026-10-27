package widgets

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/dropdown"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// SelectRow is a labeled dropdown kept under key; it opens as a bottom
// sheet on phones.
func SelectRow(key, label, hint string, options []string, selected int, onChange func(int)) El {
	sel := giode.Keyed("select", key+"|"+strings.Join(options, "|"), func() *dropdown.Select {
		st := InputStyles()
		st.Padding = giode.SymmetricInset(12, 8)
		st.FontSize = theme.SizeFootnote
		return giode.Input.Select(st, options...).Sheet(giode.SheetCompact)
	})
	if sel.Selected() != selected {
		sel.SetSelected(selected)
	}
	sel.OnChange(onChange)
	head := Row(St{Width: -1, Gap: 12},
		giode.Box(St{FlexGrow: 1}, Text(label, theme.SizeCallout, theme.T.Text, giode.FontWeightMedium)),
		giode.Box(St{Width: 150}, sel))
	if hint == "" {
		return head
	}
	return Col(St{Gap: 4, Width: -1}, head, Text(hint, theme.SizeCaption, theme.T.Muted))
}
