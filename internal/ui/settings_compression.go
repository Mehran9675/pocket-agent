package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// The limits offered, in thousands of tokens; 0 is automatic.
var (
	cloudLimits = []int{32, 64, 128, 200, 500, 1000}
	localLimits = []int{0, 2, 4, 8, 16, 32}
)

// tokensLabel shows a limit given in thousands of tokens: 128K, 1M.
func tokensLabel(k int) string {
	if k >= 1000 && k%1000 == 0 {
		return fmt.Sprintf("%dM", k/1000)
	}
	return fmt.Sprintf("%dK", k)
}

func (a *App) compressionBody() widgets.El {
	cfg := &a.settings.Compression
	chips := func(key string, limits []int, cur *int) widgets.El {
		var els []widgets.El
		for _, n := range limits {
			n := n
			label := tr("Auto")
			if n > 0 {
				label = tokensLabel(n)
			}
			els = append(els, widgets.Chip(fmt.Sprintf("comp/%s/%d", key, n), label, "", *cur == n, func() {
				*cur = n
				a.saveSettings()
			}))
		}
		return giode.Row(widgets.St{Gap: 8, Wrap: giode.WrapWrap}, els...)
	}
	if cfg.CloudLimit <= 0 {
		cfg.CloudLimit = 128
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
		kids = append(kids,
			widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("Cloud models")),
				widgets.Card(widgets.St{Gap: 10}, chips("cloud", cloudLimits, &cfg.CloudLimit),
					widgets.Text(tr("Cloud models have large context windows. A higher limit keeps more detail; a lower one saves tokens and money."), theme.SizeFootnote, theme.T.Muted))),
			widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("On-device models")),
				widgets.Card(widgets.St{Gap: 10}, chips("local", localLimits, &cfg.LocalLimit),
					widgets.Text(tr("Auto summarizes at 60% of the loaded model's context window. A set limit never goes past that."), theme.SizeFootnote, theme.T.Muted))))
	} else {
		kids = append(kids, widgets.Text(tr("Off: conversations are sent in full. On-device models drop their oldest messages when the context window fills."), theme.SizeFootnote, theme.T.Muted))
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1}, kids...)
}

func compressionSummary(a *App) string {
	cfg := a.settings.Compression
	if !cfg.Enabled {
		return tr("Off")
	}
	local := tr("Auto")
	if cfg.LocalLimit > 0 {
		local = tokensLabel(cfg.LocalLimit)
	}
	return tr("Cloud {cloud} · on-device {local}", "cloud", tokensLabel(cfg.Cloud()/1000), "local", local)
}
