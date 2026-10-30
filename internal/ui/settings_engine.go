package ui

import (
	"fmt"
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) engineBody() widgets.El {
	ls := &a.settings.Local
	unload := func() {
		a.engine.Unload()
		a.toast(giode.ToastInfo, tr("Model unloaded"))
	}
	status := tr("No model loaded")
	if p := a.engine.Loaded(); p != "" {
		status = tr("Loaded: {model}", "model", lastPathElem(p))
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
				return widgets.Button("eng/unload", tr("Unload model"), "", widgets.BtnSecondary, unload)
			})),
		a.contextPicker(ls),
		widgets.Group(
			widgets.ListRow("eng/sampling", "sliders-horizontal", "#f59e0b", tr("Sampling"), samplingSummary(ls), nil, func() { a.navigate("settings/local/sampling") }),
			widgets.ListRow("eng/perf", "gauge", "#34d399", tr("Performance & memory"), perfSummary(ls), nil, func() { a.navigate("settings/local/perf") }),
		),
	)
}

// samplingSummary describes the sampling settings in a line.
func samplingSummary(ls *store.LocalSettings) string {
	g := ls.Sampling
	s := fmt.Sprintf(tr("Temperature %.2f"), ls.Temperature)
	switch {
	case g.Mirostat == 2:
		s += tr(" · Mirostat")
	case g.DRYMultiplier > 0:
		s += tr(" · DRY")
	}
	return s + " · " + trn("up to {n} tokens", ls.MaxTokens)
}

// perfSummary describes the load options in a line.
func perfSummary(ls *store.LocalSettings) string {
	lo := ls.Load
	k, v := kvLabels[indexOr(kvTypes, lo.CacheTypeK, 0)], kvLabels[indexOr(kvTypes, lo.CacheTypeV, 0)]
	s := tr("KV cache {type}", "type", k)
	if k != v {
		s += "/" + v
	}
	s += tr(" · flash attention {mode}", "mode", strings.ToLower(tr(flashLabels[indexOr(flashModes, lo.FlashAttn, 0)])))
	if ls.GPULayers == 0 {
		s += tr(" · CPU only")
	}
	return s
}

// contextOptions are the context windows offered, in tokens; -1 is the
// model's own maximum.
var contextOptions = []struct {
	tokens int
	label  string
}{
	{2048, "2K"}, {4096, "4K"}, {8192, "8K"}, {16384, "16K"}, {32768, "32K"},
	{65536, "64K"}, {131072, "128K"}, {-1, key("Model max")},
}

// contextPicker chooses the on-device context window.
func (a *App) contextPicker(ls *store.LocalSettings) widgets.El {
	var chips []widgets.El
	for _, o := range contextOptions {
		o := o
		chips = append(chips, widgets.Chip(fmt.Sprintf("eng/ctx/%d", o.tokens), tr(o.label), "", ls.ContextSize == o.tokens, func() {
			ls.ContextSize = o.tokens
			a.saveSettings()
			a.engine.Unload()
		}))
	}
	hint := tr("Longer context remembers more of the chat but needs more memory and a slower first reply.")
	if ls.ContextSize > 16384 || ls.ContextSize < 0 {
		hint = tr("Large windows take a lot of memory - several GB at 64K and up. If a model fails to load or the app closes, pick a smaller window. Models never go past the context they were trained for.")
	}
	return widgets.Col(widgets.St{Gap: 8, Width: -1},
		widgets.Text(tr("Context window"), theme.SizeCallout, theme.T.Text, giode.FontWeightMedium),
		giode.Row(widgets.St{Gap: 8, Wrap: giode.WrapWrap, Width: -1}, chips...),
		widgets.Text(hint, theme.SizeCaption, theme.T.Muted))
}
