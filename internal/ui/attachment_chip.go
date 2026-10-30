package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// attachmentChip shows an attachment, with a remove button when
// onRemove is set.
func (a *App) attachmentChip(key string, att store.Attachment, loading bool, onRemove func()) widgets.El {
	var lead widgets.El
	if att.Kind == "image" && !loading {
		if img := a.composer.thumb(att); img != nil {
			lead = giode.Image(img, widgets.St{Width: 40, Height: 40, Fit: giode.FitCover, BorderRadius: 8})
		}
	}
	if lead == nil {
		icon := "file-text"
		if att.Kind == "image" {
			icon = "image"
		}
		if loading {
			lead = giode.Box(widgets.St{Width: 40, Height: 40, BorderRadius: 8, Background: theme.T.Surface2, Align: giode.AlignCenter, Justify: giode.JustifyCenter},
				giode.Spinner(widgets.St{Width: 18, Height: 18, Color: theme.T.Primary}))
		} else {
			lead = widgets.IconTile(icon, theme.T.Primary, 40)
		}
	}
	detail := tools.HumanBytes(att.Size)
	switch {
	case loading:
		detail = tr("Reading…")
	case att.Pages > 0:
		detail = trn("{n} pages", att.Pages) + " · " + detail
	case att.Chars > 0:
		detail = fmt.Sprintf(tr("%s chars"), compactNum(att.Chars))
	}
	kids := []widgets.El{lead, widgets.Col(widgets.St{Gap: 2, MaxWidth: 150}, widgets.Line(att.Name, theme.SizeFootnote, theme.T.Text, giode.FontWeightMedium), widgets.Line(detail, theme.SizeCaption, theme.T.Muted))}
	if onRemove != nil {
		kids = append(kids, widgets.IconButton(key+"/rm", "x", tr("Remove attachment"), 14, theme.T.Muted, onRemove))
	}
	return widgets.Row(widgets.St{Gap: 10, Padding: giode.Inset{Left: 6, Top: 6, Bottom: 6, Right: 8}, BorderRadius: 14, Background: theme.T.Surface, BorderWidth: 1, BorderColor: theme.T.Border}, kids...)
}
