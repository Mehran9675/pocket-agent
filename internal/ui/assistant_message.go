package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) assistantMessage(c *store.Conversation, m *store.Message, last bool) widgets.El {
	var kids []widgets.El
	if m.Research != nil {
		kids = append(kids, a.researchCard(m))
	}
	if len(m.Plan) > 0 {
		kids = append(kids, a.planCard(m.Plan, m.Streaming))
	}
	// Group consecutive tool calls so a burst of searches reads as one
	// step.
	for i := 0; i < len(m.Blocks); i++ {
		b := m.Blocks[i]
		switch b.Kind {
		case store.BlockReasoning:
			thinking := m.Streaming && i == len(m.Blocks)-1
			kids = append(kids, a.reasoningBlock(m, i, b.Text, thinking))
		case store.BlockText:
			kids = append(kids, a.markdown(m.ID+fmt.Sprint(i), b.Text))
		case store.BlockTool:
			var group []*store.ToolRun
			for ; i < len(m.Blocks) && m.Blocks[i].Kind == store.BlockTool; i++ {
				group = append(group, m.Blocks[i].Tool)
			}
			i--
			kids = append(kids, a.toolGroup(c, m, group))
		}
	}
	if m.Streaming && (len(m.Blocks) == 0 || m.Blocks[len(m.Blocks)-1].Kind == store.BlockTool) && m.Research == nil && !awaitingApproval(m) {
		label := "Thinking"
		if a.loadingLLM != "" {
			label = "Loading model into memory"
		}
		kids = append(kids, widgets.Row(widgets.St{Gap: 10, Padding: giode.Inset{Top: 2}},
			giode.Spinner(widgets.St{Width: 16, Height: 16, Color: theme.T.Primary}),
			widgets.Text(label, theme.SizeCallout, theme.T.Muted)))
	}
	if files := m.Files(); len(files) > 0 && !m.Streaming {
		kids = append(kids, a.filesCard(c, files))
	}
	if srcs := m.Sources(); len(srcs) > 0 && !m.Streaming {
		kids = append(kids, a.sourcesRow(m.ID, srcs))
	}
	if m.Error != "" {
		kids = append(kids, widgets.Row(widgets.St{
			Gap: 10, Padding: giode.UniformInset(12), BorderRadius: 12, Width: -1, Align: giode.AlignStart,
			Background: theme.T.DangerSoft, BorderWidth: 1, BorderColor: theme.Tint(theme.T.Danger, 0.3),
		},
			widgets.Icon("triangle-alert", 18, theme.T.Danger),
			giode.Box(widgets.St{FlexGrow: 1}, widgets.Text(m.Error, theme.SizeCallout, theme.T.Text)),
			widgets.When(last && !a.isRunning(c), func() widgets.El {
				return widgets.Button("retry/"+m.ID, "Retry", "rotate-ccw", widgets.BtnSecondary, func() { a.regenerate(c) })
			}),
		))
	}
	if m.Stopped {
		kids = append(kids, widgets.Row(widgets.St{Gap: 6}, widgets.Icon("square", 13, theme.T.Faint), widgets.Text("Stopped", theme.SizeFootnote, theme.T.Faint)))
	}
	if !m.Streaming {
		kids = append(kids, a.messageFooter(c, m, last))
	}
	return widgets.Col(widgets.St{Gap: 12, Width: -1}, kids...)
}

// awaitingApproval reports whether a tool call in m waits for the user.
func awaitingApproval(m *store.Message) bool {
	for _, b := range m.Blocks {
		if b.Tool != nil && b.Tool.Status == store.ToolPending {
			return true
		}
	}
	return false
}
