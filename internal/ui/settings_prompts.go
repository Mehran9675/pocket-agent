package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/agent"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) promptsBody() widgets.El {
	var rows []widgets.El
	for _, d := range agent.PromptDefs {
		id := string(d.ID)
		sub := tr("Default")
		if strings.TrimSpace(a.settings.Prompts[id]) != "" {
			sub = tr("Customized")
		}
		rows = append(rows, widgets.ListRow("prompt/"+id, "square-terminal", "#a78bfa", tr(d.Title), sub, nil, func() { a.navigate("settings/prompt/" + id) }))
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1},
		widgets.Text(tr("The instructions Pocket Agent gives models behind the scenes. Rewrite them to change how it answers, works and researches; each can go back to its default."), theme.SizeFootnote, theme.T.Muted),
		widgets.Group(rows...),
		widgets.Text(tr("Agents' own instructions are edited with each agent; custom instructions and per-chat instructions are added on top."), theme.SizeFootnote, theme.T.Muted))
}

func promptTitle(id string) string {
	for _, d := range agent.PromptDefs {
		if string(d.ID) == id {
			return tr(d.Title)
		}
	}
	return id
}

// promptBody edits one system prompt.
func (a *App) promptBody(id string) widgets.El {
	var def agent.PromptDef
	for _, d := range agent.PromptDefs {
		if string(d.ID) == id {
			def = d
		}
	}
	key := "prompt/edit/" + id
	in := a.input(key, "", agent.Prompts(a.settings.Prompts).Get(def.ID), func(i *input.Input) { i.Multiline(true) })
	set := func(text string) {
		if a.settings.Prompts == nil {
			a.settings.Prompts = map[string]string{}
		}
		text = strings.TrimSpace(text)
		if text == "" || text == def.Default {
			delete(a.settings.Prompts, id)
		} else {
			a.settings.Prompts[id] = text
		}
		a.saveSettings()
	}
	return widgets.Col(widgets.St{Gap: 14, Width: -1},
		widgets.Text(tr(def.Help), theme.SizeFootnote, theme.T.Muted),
		giode.Box(widgets.St{Width: -1, MinHeight: 260}, in),
		widgets.Row(widgets.St{Gap: 10, Width: -1, Wrap: giode.WrapWrap}, widgets.Spacer(),
			widgets.Button("prompt/reset", tr("Reset to default"), "rotate-ccw", widgets.BtnSecondary, func() {
				in.SetText(def.Default)
				set("")
				a.toast(giode.ToastSuccess, tr("Prompt reset"))
			}),
			widgets.Button("prompt/save", tr("Save"), "check", widgets.BtnPrimary, func() {
				set(in.Text())
				a.toast(giode.ToastSuccess, tr("Prompt saved"))
			})),
	)
}
