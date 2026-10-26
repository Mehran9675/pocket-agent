package markdown

import (
	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (d *Doc) codeBlock(key string, b mdBlock) widgets.El {
	lang := b.lang
	if lang == "" {
		lang = "code"
	}
	code := b.code
	header := widgets.Row(widgets.St{Padding: giode.Inset{Left: 14, Right: 6, Top: 4, Bottom: 4}, Width: -1, Background: theme.T.Surface2},
		widgets.Text(lang, theme.SizeCaption, theme.T.Muted, pr.FontWeightMedium),
		widgets.Spacer(),
		widgets.Tap(d.Key+key+"/copy", "Copy code", func(s giode.PressState) widgets.El {
			return widgets.Row(widgets.St{Gap: 6, Padding: giode.SymmetricInset(8, 6), BorderRadius: 8, Background: widgets.HoverBg(s, "transparent")},
				widgets.Icon("copy", 14, theme.T.Muted), widgets.Text("Copy", theme.SizeCaption, theme.T.Muted))
		}, func() {
			if d.OnCopy != nil {
				d.OnCopy(code)
			}
		}),
	)
	body := giode.Box(widgets.St{Padding: giode.SymmetricInset(14, 12), Width: -1},
		widgets.CodeText(code, 13, theme.T.Text))
	return giode.Box(widgets.St{
		FlexDirection: giode.FlexDirectionColumn, Width: -1, BorderRadius: 12, Overflow: giode.OverflowHidden,
		Background: theme.T.CodeBg, BorderWidth: 1, BorderColor: theme.T.CodeBorder,
	}, header, body)
}
