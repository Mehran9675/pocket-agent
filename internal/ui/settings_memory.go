package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) memoryBody() widgets.El {
	facts := a.st.All()
	var rows []widgets.El
	for i, f := range facts {
		f := f
		rows = append(rows, widgets.Row(widgets.St{Gap: 10, Padding: giode.Inset{Left: 14, Right: 6, Top: 6, Bottom: 6}, Width: -1},
			giode.Box(widgets.St{FlexGrow: 1}, widgets.Text(f, theme.SizeCallout, theme.T.Text)),
			widgets.IconButton(fmt.Sprintf("mem/rm/%d", i), "x", "Forget", 16, theme.T.Muted, func() { a.st.Forget(f) })))
	}
	if len(rows) == 0 {
		rows = append(rows, giode.Box(widgets.St{Padding: giode.UniformInset(14)}, widgets.Text("Nothing yet. Tell Pocket Agent “remember that…” or let it save useful facts as you chat.", theme.SizeFootnote, theme.T.Muted)))
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1},
		widgets.Group(widgets.ListRow("mem/on", "brain", "#f59e0b", "Use memory", "Let models save and recall facts about you across chats", widgets.Toggle("mem/t", a.settings.MemoryEnabled, func(v bool) {
			a.settings.MemoryEnabled = v
			a.saveSettings()
		}), nil)),
		widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(fmt.Sprintf("Memories · %d", len(facts))), widgets.Card(widgets.St{Padding: giode.UniformInset(4), Gap: 0}, rows...)),
		widgets.When(len(facts) > 0, func() widgets.El {
			return widgets.Button("mem/clear", "Clear all memories", "trash-2", widgets.BtnDanger, func() {
				a.ask("Clear all memories?", "Pocket Agent will forget everything it saved about you.", "Clear", true, a.st.ClearMemory)
			})
		}),
	)
}
