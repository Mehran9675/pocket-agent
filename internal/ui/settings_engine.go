package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) engineBody() widgets.El {
	ls := &a.settings.Local
	ctxOpts := []int{2048, 4096, 8192, 16384}
	ctxIdx := 1
	for i, v := range ctxOpts {
		if v == ls.ContextSize {
			ctxIdx = i
		}
	}
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
	if !a.engine.Available() {
		return widgets.Card(widgets.St{},
			widgets.Row(widgets.St{Gap: 10}, widgets.Icon("triangle-alert", 18, theme.T.Warning), widgets.Text("On-device engine unavailable", theme.SizeCallout, theme.T.Text, giode.FontWeightBold)),
			widgets.Text(a.engine.Problem(), theme.SizeFootnote, theme.T.Text2))
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1},
		widgets.Card(widgets.St{},
			widgets.Row(widgets.St{Gap: 10}, widgets.IconTile("cpu", "#34d399", 34), widgets.Col(widgets.St{Gap: 2}, widgets.Text("llama.cpp", theme.SizeCallout, theme.T.Text, giode.FontWeightBold), widgets.Text(status, theme.SizeFootnote, theme.T.Muted))),
			widgets.When(a.engine.Loaded() != "", func() widgets.El {
				return widgets.Button("eng/unload", "Unload model", "", widgets.BtnSecondary, unload)
			})),
		widgets.Col(widgets.St{Gap: 8, Width: -1}, widgets.Text("Context window", theme.SizeCallout, theme.T.Text, giode.FontWeightMedium),
			widgets.Text("Longer context remembers more of the chat but uses more memory.", theme.SizeCaption, theme.T.Muted),
			widgets.Segmented("eng/ctx", []string{"2K", "4K", "8K", "16K"}, nil, ctxIdx, func(i int) {
				ls.ContextSize = ctxOpts[i]
				a.saveSettings()
				a.engine.Unload()
			})),
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
