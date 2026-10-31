package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// compressionLimits are the limits offered, in thousands of tokens.
var compressionLimits = []int{8, 16, 32, 64, 128, 256}

func (a *App) compressionBody() widgets.El {
	cfg := &a.settings.Compression
	var chips []widgets.El
	for _, n := range compressionLimits {
		n := n
		chips = append(chips, widgets.Chip(fmt.Sprintf("comp/%d", n), fmt.Sprintf("%dK", n), "", cfg.Limit == n, func() {
			cfg.Limit = n
			a.saveSettings()
		}))
	}
	kids := []widgets.El{
		widgets.Group(widgets.ListRow("comp/on", "layers", "#38bdf8", tr("Summarize long conversations"),
			tr("Condense earlier messages instead of dropping them"), widgets.Toggle("comp/t", cfg.Enabled, func(v bool) {
				cfg.Enabled = v
				a.saveSettings()
			}), nil)),
		widgets.Text(tr("When a conversation's context grows past the limit, the earlier messages are summarized and the summary is sent in their place; the latest exchange is always kept word for word. Agents also condense their earlier steps during long tasks, which keeps them from using up their token budget."), theme.SizeFootnote, theme.T.Muted),
	}
	if cfg.Enabled {
		kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("Limit")),
			widgets.Card(widgets.St{Gap: 10},
				giode.Row(widgets.St{Gap: 8, Wrap: giode.WrapWrap}, chips...),
				widgets.Text(tr("Lower limits save tokens and money; higher ones keep more detail. On-device models summarize before their context window fills, whatever the limit."), theme.SizeFootnote, theme.T.Muted))))
	} else {
		kids = append(kids, widgets.Text(tr("Off: conversations are sent in full. On-device models drop their oldest messages when the context window fills."), theme.SizeFootnote, theme.T.Muted))
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1}, kids...)
}

func compressionSummary(a *App) string {
	if !a.settings.Compression.Enabled {
		return tr("Off")
	}
	return tr("Over {limit} tokens", "limit", fmt.Sprintf("%dK", a.settings.Compression.Tokens()/1000))
}
