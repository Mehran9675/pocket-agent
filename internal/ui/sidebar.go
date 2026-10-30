package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

type sidebarView struct {
	a      *App
	search *input.Input
	// current holds each conversation row built this frame, for the
	// swipeable wrapping it.
	current map[string]widgets.El
}

func newSidebar(a *App) *sidebarView {
	s := &sidebarView{a: a}
	s.search = giode.Input.Text(tr("Search"), widgets.St{
		Background: "theme.surface", BorderColor: "transparent", BorderRadius: 10,
		Padding: giode.Inset{Left: 36, Right: 10, Top: 9, Bottom: 9}, Color: "theme.text", FontSize: theme.SizeCallout,
	}).Label(tr("Search conversations"))
	return s
}

func (s *sidebarView) view(inDrawer bool) widgets.El {
	a := s.a
	top := widgets.Row(widgets.St{Width: -1, Padding: giode.Inset{Left: 16, Right: 8, Top: 12, Bottom: 8}, Gap: 10},
		widgets.LogoMark(30),
		widgets.Text("Pocket Agent", theme.SizeHeadline, theme.T.Text, giode.FontWeightSemiBold),
		widgets.Spacer(),
		widgets.IconButton("side/new", "square-pen", tr("New chat"), 19, theme.T.Text2, func() { a.newChat(store.ModeChat, "") }),
	)
	searchBox := giode.Box(widgets.St{Width: -1, Padding: giode.SymmetricInset(12, 4)},
		giode.Box(widgets.St{Width: -1},
			s.search,
			giode.Box(widgets.St{Position: giode.PositionAbsolute, Left: 11, Top: 9}, widgets.Icon("search", 16, theme.T.Muted))))

	nav := func(key, icon, label string, active bool, onClick func()) widgets.El {
		return widgets.Tap("side/nav/"+key, label, func(st giode.PressState) widgets.El {
			bg := widgets.HoverBg(st, "transparent")
			fg := theme.T.Text2
			if active {
				bg, fg = theme.T.Hover, theme.T.Text
			}
			return widgets.Row(widgets.St{Gap: 12, Padding: giode.SymmetricInset(12, 9), BorderRadius: 10, Background: bg, Width: -1},
				widgets.Icon(icon, 18, fg), widgets.Text(label, theme.SizeCallout, fg, giode.FontWeightMedium))
		}, onClick)
	}
	navs := widgets.Col(widgets.St{Gap: 1, Width: -1, Padding: giode.SymmetricInset(8, 4)},
		nav("agents", "bot", tr("Agents"), strings.HasPrefix(a.route, "agents"), func() { a.navigateRoot("agents") }),
		nav("research", "telescope", tr("Deep research"), false, func() { a.newChat(store.ModeResearch, "") }),
		nav("library", "hard-drive", tr("On-device models"), a.route == "library", func() { a.navigateRoot("library") }),
	)

	list := a.scroll("side/list", func() widgets.El { return s.conversationList() })
	bottom := widgets.Col(widgets.St{Width: -1, Padding: giode.SymmetricInset(8, 8)},
		widgets.HDivider(),
		widgets.Gap(0, 6),
		widgets.Tap("side/settings", tr("Settings"), func(st giode.PressState) widgets.El {
			active := strings.HasPrefix(a.route, "settings")
			bg := widgets.HoverBg(st, "transparent")
			if active {
				bg = theme.T.Hover
			}
			name, _ := a.modelLabel(a.settings.DefaultModel)
			return widgets.Row(widgets.St{Gap: 12, Padding: giode.SymmetricInset(12, 9), BorderRadius: 10, Background: bg, Width: -1},
				widgets.Icon("settings", 18, theme.T.Text2),
				widgets.Col(widgets.St{Gap: 1, FlexGrow: 1},
					widgets.Text(tr("Settings"), theme.SizeCallout, theme.T.Text, giode.FontWeightMedium),
					widgets.Line(name, theme.SizeCaption, theme.T.Muted)))
		}, func() { a.navigateRoot("settings") }),
	)
	return widgets.Col(widgets.St{Width: -1, Height: -1},
		top, searchBox, navs,
		giode.Box(widgets.St{Padding: giode.SymmetricInset(16, 6), Width: -1}, widgets.HDivider()),
		giode.Box(widgets.St{FlexGrow: 1, Width: -1}, list),
		bottom,
	)
}
