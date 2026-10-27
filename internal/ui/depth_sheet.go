package ui

import (
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) openDepthSheet() {
	depth := a.draft.depth
	if a.current != nil {
		depth = a.current.Depth
	}
	set := func(d store.ResearchDepth) {
		a.closeSheet()
		if a.current != nil {
			a.current.Depth = d
			a.st.SaveConversation(a.current)
		}
		a.draft.depth = d
	}
	depthRow := func(d store.ResearchDepth, title, sub string) widgets.El {
		var tr widgets.El
		if d == depth {
			tr = widgets.Icon("check", 18, theme.T.Primary)
		}
		return widgets.ListRow("depth/"+string(d), "gauge", "#38bdf8", title, sub, tr, func() { set(d) })
	}
	a.openSheet("depth", "Research depth", func() widgets.El {
		return widgets.Group(
			depthRow(store.DepthQuick, "Quick", "4 questions · 1 round · a few minutes"),
			depthRow(store.DepthStandard, "Standard", "6 questions · up to 3 rounds · about 20 sources"),
			depthRow(store.DepthDeep, "Deep", "8 questions · up to 5 rounds · 40+ sources and a long report"),
		)
	})
}
