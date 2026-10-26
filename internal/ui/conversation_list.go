package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (s *sidebarView) conversationList() widgets.El {
	a := s.a
	q := strings.ToLower(strings.TrimSpace(s.search.Text()))
	var kids []widgets.El
	last := ""
	count := 0
	for _, c := range a.convs {
		if q != "" && !strings.Contains(strings.ToLower(c.Title), q) && !strings.Contains(strings.ToLower(c.Preview()), q) {
			continue
		}
		bucket := dayBucket(c.Updated)
		if c.Pinned {
			bucket = "Pinned"
		}
		if bucket != last {
			last = bucket
			kids = append(kids, giode.Box(widgets.St{Padding: giode.Inset{Left: 12, Top: 14, Bottom: 4}},
				widgets.Text(bucket, theme.SizeCaption, theme.T.Muted, giode.FontWeightBold)))
		}
		kids = append(kids, s.convRow(c))
		count++
	}
	if count == 0 {
		msg := "Your conversations will appear here."
		if q != "" {
			msg = "No conversations match “" + q + "”."
		}
		kids = append(kids, giode.Box(widgets.St{Padding: giode.UniformInset(16)}, widgets.Text(msg, theme.SizeFootnote, theme.T.Muted)))
	}
	return widgets.Col(widgets.St{Width: -1, Padding: giode.Inset{Left: 8, Right: 8, Bottom: 12}}, kids...)
}
