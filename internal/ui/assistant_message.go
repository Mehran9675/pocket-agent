package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// assistantRows lays a reply out as rows for the message list: cards,
// each tool group, and long text and thinking in chunks, so only the
// parts on screen are laid out (see widgets.VirtualColumn).
func (a *App) assistantRows(c *store.Conversation, m *store.Message, last bool) []msgRow {
	running := a.isRunning(c)
	// state changes with anything that may change a non-text row.
	state := fmt.Sprint(m.Version, last, running, a.expandedSig(m), a.loadingLLM != "", a.textEnv())
	var rows []msgRow
	add := func(key, token string, build func() widgets.El) {
		rows = append(rows, msgRow{Key: m.ID + "/" + key, Token: token, Gap: 12, Build: build})
	}
	if m.Research != nil {
		add("research", state, func() widgets.El { return a.researchCard(m) })
	}
	if len(m.Plan) > 0 {
		add("plan", state, func() widgets.El { return a.planCard(m.Plan, m.Streaming) })
	}
	size := a.textSize()
	// Group consecutive tool calls so a burst of searches reads as one
	// step.
	for i := 0; i < len(m.Blocks); i++ {
		b := m.Blocks[i]
		switch b.Kind {
		case store.BlockReasoning:
			a.reasoningRows(m, i, b.Text, m.Streaming && i == len(m.Blocks)-1, add)
		case store.BlockText:
			for j, chunk := range splitMarkdown(b.Text, chunkSize) {
				key := fmt.Sprintf("t%d/%d", i, j)
				add(key, textToken(chunk)+a.textEnv(), func() widgets.El { return a.markdownChunk(m.ID+"/"+key, chunk, size) })
			}
		case store.BlockTool:
			start := i
			var group []*store.ToolRun
			for ; i < len(m.Blocks) && m.Blocks[i].Kind == store.BlockTool; i++ {
				group = append(group, m.Blocks[i].Tool)
			}
			i--
			add(fmt.Sprintf("tools%d", start), state, func() widgets.El { return a.toolGroup(c, m, group) })
		}
	}
	if m.Streaming && (len(m.Blocks) == 0 || m.Blocks[len(m.Blocks)-1].Kind == store.BlockTool) && m.Research == nil && !awaitingApproval(m) {
		label := "Thinking"
		if a.loadingLLM != "" {
			label = "Loading model into memory"
		}
		add("spinner", label, func() widgets.El {
			return widgets.Row(widgets.St{Gap: 10, Padding: giode.Inset{Top: 2}},
				giode.Spinner(widgets.St{Width: 16, Height: 16, Color: theme.T.Primary}),
				widgets.Text(label, theme.SizeCallout, theme.T.Muted))
		})
	}
	if files := m.Files(); len(files) > 0 && !m.Streaming {
		add("files", state, func() widgets.El { return a.filesCard(c, files) })
	}
	if srcs := m.Sources(); len(srcs) > 0 && !m.Streaming {
		add("sources", state, func() widgets.El { return a.sourcesRow(m.ID, srcs) })
	}
	if m.Error != "" {
		add("error", state, func() widgets.El {
			return widgets.Row(widgets.St{
				Gap: 10, Padding: giode.UniformInset(12), BorderRadius: 12, Width: -1, Align: giode.AlignStart,
				Background: theme.T.DangerSoft, BorderWidth: 1, BorderColor: theme.Tint(theme.T.Danger, 0.3),
			},
				widgets.Icon("triangle-alert", 18, theme.T.Danger),
				giode.Box(widgets.St{FlexGrow: 1}, widgets.Text(m.Error, theme.SizeCallout, theme.T.Text)),
				widgets.When(last && !running, func() widgets.El {
					return widgets.Button("retry/"+m.ID, "Retry", "rotate-ccw", widgets.BtnSecondary, func() { a.regenerate(c) })
				}),
			)
		})
	}
	if m.Stopped {
		add("stopped", state, func() widgets.El {
			return widgets.Row(widgets.St{Gap: 6}, widgets.Icon("square", 13, theme.T.Faint), widgets.Text("Stopped", theme.SizeFootnote, theme.T.Faint))
		})
	}
	if !m.Streaming {
		add("footer", state, func() widgets.El { return a.messageFooter(c, m, last) })
	}
	return rows
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
