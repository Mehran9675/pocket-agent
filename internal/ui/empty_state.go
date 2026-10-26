package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// emptyState is the new-chat screen.
func (v *chatView) emptyState(compact bool) widgets.El {
	a := v.a
	if !a.hasAnyModel() {
		return a.scroll("empty/onboard", func() widgets.El { return v.onboarding(compact) })
	}
	mode, agentID := a.draft.mode, a.draft.agentID
	if a.current != nil {
		mode, agentID = a.current.Mode, a.current.AgentID
	}
	modeIdx := map[store.Mode]int{store.ModeChat: 0, store.ModeAgent: 1, store.ModeResearch: 2}[mode]

	title, subtitle := greeting(), "What can I help you with?"
	var suggestions []string
	accent := theme.T.Primary
	icon := ""
	switch mode {
	case store.ModeAgent:
		if ag := a.agentByID(agentID); ag != nil {
			title, subtitle = ag.Name, ag.Description
			suggestions = ag.Starters
			accent, icon = ag.Color, ag.Icon
		}
	case store.ModeResearch:
		title, subtitle = "Deep research", "Ask a question. Pocket Agent plans, searches and reads the web, then writes a cited report."
		suggestions = []string{
			"How do the leading on-device AI chips compare in 2026?",
			"What does the latest research say about microplastics and health?",
			"Compare the costs of heat pumps vs. gas boilers in Europe",
		}
		accent, icon = "#38bdf8", "telescope"
	default:
		suggestions = []string{
			"Explain quantum error correction like I'm an engineer",
			"Draft a polite follow-up email after a job interview",
			"Plan a high-protein vegetarian week of meals",
			"Write a regex that validates ISO 8601 dates",
		}
	}
	var mark widgets.El = widgets.LogoMark(52)
	if icon != "" {
		mark = widgets.IconTile(icon, accent, 52)
	}
	var sugg []widgets.El
	for i, s := range suggestions {
		s := s
		sugg = append(sugg, widgets.Tap(fmt.Sprintf("sugg/%d/%s", i, s), s, func(st giode.PressState) widgets.El {
			return widgets.Row(widgets.St{
				Gap: 12, Padding: giode.SymmetricInset(14, 12), BorderRadius: 14, Width: -1,
				Background: widgets.HoverBg(st, theme.T.Surface), BorderWidth: 1, BorderColor: theme.T.Border,
			}, widgets.Icon("arrow-up", 16, theme.T.Muted), giode.Box(widgets.St{FlexGrow: 1}, widgets.Text(s, theme.SizeCallout, theme.T.Text2)))
		}, func() {
			a.composer.in.SetText(s)
			a.composer.submit()
		}))
	}
	content := widgets.Col(widgets.St{Width: -1, Gap: 18, Align: giode.AlignCenter, Padding: giode.Inset{Top: 24, Bottom: 24}},
		mark,
		widgets.Col(widgets.St{Gap: 6, Align: giode.AlignCenter},
			widgets.Text(title, theme.SizeDisplay, theme.T.Text, giode.FontWeightBold),
			giode.Text(subtitle, widgets.St{FontSize: theme.SizeBody, Color: theme.T.Muted, TextAlign: giode.TextAlignCenter}),
		),
		giode.Box(widgets.St{Width: -1, Padding: giode.Inset{Top: 4}}, widgets.Centered(420,
			widgets.Segmented("empty/mode", []string{"Chat", "Agent", "Research"}, []string{"message-square", "bot", "telescope"}, modeIdx, func(i int) {
				m := []store.Mode{store.ModeChat, store.ModeAgent, store.ModeResearch}[i]
				a.newChat(m, "")
			}))),
		widgets.Col(widgets.St{Width: -1, Gap: 8, Padding: giode.Inset{Top: 6}}, sugg...),
	)
	return a.scroll("empty/"+string(mode), func() widgets.El {
		return giode.Box(widgets.St{Width: -1, Padding: giode.SymmetricInset(20, 0)}, widgets.Centered(640, content))
	})
}
