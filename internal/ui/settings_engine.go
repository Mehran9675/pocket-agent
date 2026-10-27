package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) engineBody() widgets.El {
	ls := &a.settings.Local
	tempOpts := []float64{0.2, 0.7, 1.0}
	tempIdx := 1
	for i, v := range tempOpts {
		if v == ls.Temperature {
			tempIdx = i
		}
	}
	lenOpts := []int{1024, 2048, 4096}
	lenIdx := 1
	for i, v := range lenOpts {
		if v == ls.MaxTokens {
			lenIdx = i
		}
	}
	unload := func() {
		a.engine.Unload()
		a.toast(giode.ToastInfo, "Model unloaded")
	}
	status := "No model loaded"
	if p := a.engine.Loaded(); p != "" {
		status = "Loaded: " + lastPathElem(p)
	}
	card := a.engineCard()
	if !a.engine.Available() && card != nil {
		return card
	}
	if card == nil {
		card = giode.Box(widgets.St{Display: giode.DisplayNone})
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1},
		card,
		widgets.Card(widgets.St{},
			widgets.Row(widgets.St{Gap: 10}, widgets.IconTile("cpu", "#34d399", 34), widgets.Col(widgets.St{Gap: 2}, widgets.Text(a.engineLabel(), theme.SizeCallout, theme.T.Text, giode.FontWeightSemiBold), widgets.Text(status, theme.SizeFootnote, theme.T.Muted))),
			widgets.When(a.engine.Loaded() != "", func() widgets.El {
				return widgets.Button("eng/unload", "Unload model", "", widgets.BtnSecondary, unload)
			})),
		a.contextPicker(ls),
		widgets.Col(widgets.St{Gap: 8, Width: -1}, widgets.Text("Creativity", theme.SizeCallout, theme.T.Text, giode.FontWeightMedium),
			widgets.Segmented("eng/temp", []string{"Precise", "Balanced", "Creative"}, nil, tempIdx, func(i int) {
				ls.Temperature = tempOpts[i]
				a.saveSettings()
			})),
		widgets.Col(widgets.St{Gap: 8, Width: -1}, widgets.Text("Maximum reply length", theme.SizeCallout, theme.T.Text, giode.FontWeightMedium),
			widgets.Segmented("eng/len", []string{"Short", "Medium", "Long"}, nil, lenIdx, func(i int) {
				ls.MaxTokens = lenOpts[i]
				a.saveSettings()
			})),
		widgets.Group(widgets.ListRow("eng/gpu", "zap", "#34d399", "GPU acceleration", "Use Metal on iPhone and iPad when available", widgets.Toggle("eng/gpu/t", ls.GPULayers != 0, func(v bool) {
			if v {
				ls.GPULayers = -1
			} else {
				ls.GPULayers = 0
			}
			a.saveSettings()
			a.engine.Unload()
		}), nil)),
	)
}

// contextOptions are the context windows offered, in tokens; -1 is the
// model's own maximum.
var contextOptions = []struct {
	tokens int
	label  string
}{
	{2048, "2K"}, {4096, "4K"}, {8192, "8K"}, {16384, "16K"}, {32768, "32K"},
	{65536, "64K"}, {131072, "128K"}, {-1, "Model max"},
}

// contextPicker chooses the on-device context window.
func (a *App) contextPicker(ls *store.LocalSettings) widgets.El {
	var chips []widgets.El
	for _, o := range contextOptions {
		o := o
		chips = append(chips, widgets.Chip(fmt.Sprintf("eng/ctx/%d", o.tokens), o.label, "", ls.ContextSize == o.tokens, func() {
			ls.ContextSize = o.tokens
			a.saveSettings()
			a.engine.Unload()
		}))
	}
	hint := "Longer context remembers more of the chat but needs more memory and a slower first reply."
	if ls.ContextSize > 16384 || ls.ContextSize < 0 {
		hint = "Large windows take a lot of memory - several GB at 64K and up. If a model fails to load or the app closes, pick a smaller window. Models never go past the context they were trained for."
	}
	return widgets.Col(widgets.St{Gap: 8, Width: -1},
		widgets.Text("Context window", theme.SizeCallout, theme.T.Text, giode.FontWeightMedium),
		giode.Row(widgets.St{Gap: 8, Wrap: giode.WrapWrap, Width: -1}, chips...),
		widgets.Text(hint, theme.SizeCaption, theme.T.Muted))
}
