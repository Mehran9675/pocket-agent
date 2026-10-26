package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// MonoBox shows labeled monospace text, such as tool input and output.
func MonoBox(label, text string) El {
	return Col(St{Gap: 4, Width: -1},
		Text(label, theme.SizeMicro, theme.T.Faint, giode.FontWeightBold),
		giode.Box(St{Width: -1, Padding: giode.UniformInset(10), BorderRadius: 10, Background: theme.T.CodeBg, BorderWidth: 1, BorderColor: theme.T.CodeBorder},
			CodeText(text, 12, theme.T.Text2)))
}
