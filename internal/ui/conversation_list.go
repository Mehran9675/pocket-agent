package ui

import (
	"fmt"
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (s *sidebarView) conversationList() widgets.El {
	a := s.a
	q := strings.ToLower(strings.TrimSpace(s.search.Text()))
	var kids []widgets.El
	header := func(label string, trail widgets.El) widgets.El {
		if trail == nil {
			trail = giode.Box(widgets.St{})
		}
		return widgets.Row(widgets.St{Width: -1, Padding: giode.Inset{Left: 12, Right: 4, Top: 14, Bottom: 4}},
			giode.Box(widgets.St{FlexGrow: 1}, widgets.Text(label, theme.SizeCaption, theme.T.Muted, giode.FontWeightSemiBold)),
			trail)
	}

	// Folders first, unless searching: then every match is listed by date.
	inFolder := map[string]bool{}
	if q == "" {
		for _, f := range a.settings.Folders {
			inFolder[f.ID] = true
		}
		kids = append(kids, header(tr("Folders"), widgets.IconButton("side/newfolder", "folder-plus", tr("New folder"), 16, theme.T.Muted, func() { a.openFolderName(nil, nil) })))
		for _, f := range a.settings.Folders {
			var chats []*store.Conversation
			for _, c := range a.convs {
				if c.Folder == f.ID {
					chats = append(chats, c)
				}
			}
			kids = append(kids, s.folderRow(f, len(chats)))
			if !f.Collapsed {
				for _, c := range chats {
					kids = append(kids, giode.Box(widgets.St{Width: -1, Padding: giode.Inset{Left: 16}}, s.convRow(c)))
				}
			}
		}
	}

	last := ""
	count := 0
	for _, c := range a.convs {
		if q != "" && !strings.Contains(strings.ToLower(c.Title), q) && !strings.Contains(strings.ToLower(c.Preview()), q) {
			continue
		}
		if inFolder[c.Folder] {
			count++
			continue
		}
		bucket := dayBucket(c.Updated)
		if c.Pinned {
			bucket = tr("Pinned")
		}
		if bucket != last {
			last = bucket
			kids = append(kids, header(bucket, nil))
		}
		kids = append(kids, s.convRow(c))
		count++
	}
	if count == 0 {
		msg := tr("Your conversations will appear here.")
		if q != "" {
			msg = tr("No conversations match “{query}”.", "query", q)
		}
		kids = append(kids, giode.Box(widgets.St{Padding: giode.UniformInset(16)}, widgets.Text(msg, theme.SizeFootnote, theme.T.Muted)))
	}
	return widgets.Col(widgets.St{Width: -1, Padding: giode.Inset{Left: 8, Right: 8, Bottom: 12}}, kids...)
}

// folderRow is a folder in the sidebar: a tap opens or closes it, a long
// press or right click opens its menu.
func (s *sidebarView) folderRow(f *store.Folder, n int) widgets.El {
	a := s.a
	return widgets.TapAlt("side/folder/"+f.ID, f.Name, func(st giode.PressState) widgets.El {
		chev := "chevron-down"
		if f.Collapsed {
			chev = "chevron-right"
		}
		var more widgets.El = widgets.Text(fmt.Sprint(n), theme.SizeCaption, theme.T.Faint)
		if st.Hovered {
			more = widgets.IconButton("side/folder/more/"+f.ID, "ellipsis", tr("Folder options"), 16, theme.T.Muted, func() { a.openFolderMenu(f) })
		}
		return widgets.Row(widgets.St{Gap: 10, Padding: giode.Inset{Left: 12, Right: 8, Top: 4, Bottom: 4}, MinHeight: 38, BorderRadius: 10, Background: widgets.HoverBg(st, "transparent"), Width: -1},
			widgets.Icon("folder", 15, theme.T.Primary),
			giode.Box(widgets.St{FlexGrow: 1}, widgets.Line(f.Name, theme.SizeCallout, theme.T.Text2, giode.FontWeightMedium)),
			more,
			widgets.Icon(chev, 14, theme.T.Faint))
	}, func() {
		f.Collapsed = !f.Collapsed
		a.saveSettings()
	}, func() { a.openFolderMenu(f) })
}
