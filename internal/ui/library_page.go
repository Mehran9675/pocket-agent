package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/local"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) libraryPage(compact bool) widgets.El {
	return a.pageScaffold("library", tr("On-device models"), compact,
		widgets.IconButton("lib/engine", "sliders-horizontal", tr("Engine settings"), 19, theme.T.Text2, func() { a.navigate("settings/local") }),
		a.libraryBody)
}

func (a *App) libraryBody() widgets.El {
	var kids []widgets.El
	ram := "unknown"
	if a.ram > 0 {
		ram = fmt.Sprintf(tr("%.0f GB"), float64(a.ram)/(1<<30))
	}
	header := widgets.Card(widgets.St{Gap: 12, BackgroundGradient: giode.Linear(135, theme.Tint("#34d399", 0.12), theme.T.Surface), BorderColor: theme.Tint("#34d399", 0.3)},
		widgets.Row(widgets.St{Gap: 12, Width: -1},
			widgets.IconTile("shield-check", "#34d399", 40),
			widgets.Col(widgets.St{Gap: 2, FlexGrow: 1},
				widgets.Text(tr("Private by design"), theme.SizeHeadline, theme.T.Text, giode.FontWeightSemiBold),
				widgets.Text(ifStr(isMobile(), tr("On-device models run entirely on this device. Nothing you type leaves it, and they work offline."), tr("On-device models run entirely on this computer. Nothing you type leaves it, and they work offline.")), theme.SizeFootnote, theme.T.Text2))),
		widgets.Row(widgets.St{Gap: 16},
			stat(tr("Memory"), ram), stat(tr("Engine"), a.engineLabel()),
			stat(tr("Installed"), fmt.Sprint(len(a.models.Installed())))),
	)
	kids = append(kids, header)
	if card := a.engineCard(); card != nil {
		kids = append(kids, card)
	}

	// Downloads in progress
	dls := a.models.Downloads()
	if len(dls) > 0 {
		var rows []widgets.El
		for file, d := range dls {
			file, d := file, d
			detail := fmt.Sprintf(tr("%s of %s"), tools.HumanBytes(d.Done), tools.HumanBytes(d.Total))
			if d.Speed > 0 {
				detail += fmt.Sprintf(tr(" · %s/s"), tools.HumanBytes(int64(d.Speed)))
				if rem := d.Total - d.Done; rem > 0 {
					detail += " · " + fmtDuration(time.Duration(float64(rem)/d.Speed)*time.Second)
				}
			}
			var action widgets.El = widgets.IconButton("dl/cancel/"+file, "x", tr("Cancel download"), 16, theme.T.Muted, func() { a.models.Cancel(file) })
			var bar widgets.El = giode.Progress(d.Progress(), widgets.St{Color: "#34d399", Background: theme.T.Surface2, Height: 5})
			if d.Err != nil {
				detail = d.Err.Error()
				action = widgets.Row(widgets.St{Gap: 4},
					widgets.Button("dl/retry/"+file, tr("Retry"), "", widgets.BtnSecondary, func() { a.retryDownload(file) }),
					widgets.IconButton("dl/dismiss/"+file, "x", tr("Dismiss"), 16, theme.T.Muted, func() { a.models.Dismiss(file) }))
				bar = giode.Box(widgets.St{})
			}
			rows = append(rows, widgets.Col(widgets.St{Gap: 8, Padding: giode.UniformInset(12), Width: -1},
				widgets.Row(widgets.St{Gap: 10, Width: -1},
					widgets.Icon("cloud-download", 18, "#34d399"),
					widgets.Col(widgets.St{Gap: 2, FlexGrow: 1}, widgets.Line(d.Name, theme.SizeCallout, theme.T.Text, giode.FontWeightMedium), widgets.Line(detail, theme.SizeCaption, ifStr(d.Err != nil, theme.T.Danger, theme.T.Muted))),
					action),
				bar))
		}
		kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("Downloading")), widgets.Card(widgets.St{Padding: giode.UniformInset(0), Gap: 0}, rows...)))
	}

	// Installed
	inst := a.models.Installed()
	if len(inst) > 0 {
		var rows []widgets.El
		for _, m := range inst {
			m := m
			ref := store.ModelRef{Provider: store.LocalProviderID, Model: m.File}
			inUse := a.settings.DefaultModel == ref
			var use widgets.El
			if inUse {
				use = widgets.Badge(tr("Default"), theme.T.PrimaryText, theme.T.PrimarySoft)
			} else {
				use = widgets.Button("lib/use/"+m.File, tr("Use"), "", widgets.BtnSecondary, func() {
					a.settings.DefaultModel = ref
					a.draft.model = ref
					if a.current != nil {
						a.current.Model = ref
					}
					a.saveSettings()
					a.toast(giode.ToastSuccess, tr("{name} is now the default", "name", m.Name))
				})
			}
			rows = append(rows, widgets.Row(widgets.St{Gap: 12, Padding: giode.SymmetricInset(12, 10), Width: -1},
				widgets.IconTile("cpu", "#34d399", 34),
				widgets.Col(widgets.St{Gap: 2, FlexGrow: 1}, widgets.Line(m.Name, theme.SizeCallout, theme.T.Text, giode.FontWeightMedium), widgets.Line(tools.HumanBytes(m.Size)+" · "+m.File, theme.SizeCaption, theme.T.Muted)),
				use,
				widgets.IconButton("lib/rm/"+m.File, "trash-2", tr("Delete model"), 16, theme.T.Muted, func() {
					a.ask(tr("Delete {name}?", "name", m.Name), fmt.Sprintf(tr("Frees %s. You can download it again later."), tools.HumanBytes(m.Size)), tr("Delete"), true, func() {
						if strings.HasSuffix(a.engine.Loaded(), m.File) {
							a.engine.Unload()
						}
						_ = a.models.Delete(m.File)
						if a.settings.DefaultModel == ref {
							a.settings.DefaultModel = store.ModelRef{}
							a.ensureDefaultModel()
							a.saveSettings()
						}
					})
				})))
		}
		kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("Installed")), widgets.Card(widgets.St{Padding: giode.UniformInset(0), Gap: 0}, rows...)))
	}

	// Catalog
	var cards []widgets.El
	for _, m := range local.Catalog {
		m := m
		cards = append(cards, a.catalogCard(m, dls))
	}
	kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("Recommended for mobile")),
		widgets.Col(widgets.St{Gap: 10, Width: -1}, cards...)))

	// Hugging Face
	kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(tr("More models")), a.hfSearch()))
	kids = append(kids, widgets.Button("lib/import", tr("Import a GGUF file"), "upload", widgets.BtnSecondary, a.importModel, widgets.St{Width: -1}))
	return widgets.Col(widgets.St{Gap: 18, Width: -1}, kids...)
}

func deviceNoun() string {
	if isMobile() {
		return "device"
	}
	return "computer"
}

func (a *App) retryDownload(file string) {
	a.models.Dismiss(file)
	for _, m := range local.Catalog {
		if m.File == file {
			a.models.Start(local.Installed{File: m.File, Name: m.Name, Repo: m.Repo, Size: m.Size, Context: m.Context, Reasoning: m.Reasoning}, m.URL())
			return
		}
	}
}
