package ui

import (
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/local"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) catalogCard(m local.CatalogModel, dls map[string]local.Download) widgets.El {
	installed := a.models.IsInstalled(m.File)
	dl, downloading := dls[m.File]
	fit := local.FitFor(m.MinRAM, a.ram)
	fitBadge := map[local.Fit]widgets.El{
		local.FitGreat:   widgets.Badge(tr("Runs great"), theme.T.Success, theme.T.SuccessSoft),
		local.FitOK:      widgets.Badge(tr("Good fit"), theme.T.PrimaryText, theme.T.PrimarySoft),
		local.FitTight:   widgets.Badge(tr("May be slow"), theme.T.Warning, theme.T.WarningSoft),
		local.FitUnknown: widgets.Badge(fmt.Sprintf(tr("%.0f GB+ RAM"), m.MinRAM), theme.T.Text2, theme.T.Surface2),
	}[fit]
	var action widgets.El
	switch {
	case installed:
		action = widgets.Row(widgets.St{Gap: 6}, widgets.Icon("circle-check", 16, theme.T.Success), widgets.Text(tr("Installed"), theme.SizeFootnote, theme.T.Success, giode.FontWeightMedium))
	case downloading && dl.Err != nil:
		action = widgets.Text(tr("Failed"), theme.SizeFootnote, theme.T.Danger)
	case downloading:
		action = widgets.Text(fmt.Sprintf("%.0f%%", dl.Progress()*100), theme.SizeFootnote, theme.T.Muted)
	default:
		action = widgets.Button("cat/dl/"+m.ID, tr("Get"), "download", widgets.BtnSecondary, func() {
			a.models.Start(local.Installed{File: m.File, Name: m.Name, Repo: m.Repo, Size: m.Size, Context: m.Context, Reasoning: m.Reasoning}, m.URL())
		})
	}
	tags := []widgets.El{widgets.Badge(m.Params, theme.T.Text2, theme.T.Surface2), widgets.Badge(m.Quant, theme.T.Text2, theme.T.Surface2), fitBadge}
	if m.Reasoning {
		tags = append(tags, widgets.Badge(tr("Thinking"), theme.T.Text2, theme.T.Surface2))
	}
	if m.Tools {
		tags = append(tags, widgets.Badge(tr("Tools"), theme.T.Text2, theme.T.Surface2))
	}
	return widgets.Card(widgets.St{Gap: 10},
		widgets.Row(widgets.St{Gap: 12, Width: -1, Align: giode.AlignStart},
			widgets.Avatar(m.Vendor, 36),
			widgets.Col(widgets.St{Gap: 3, FlexGrow: 1},
				widgets.Row(widgets.St{Gap: 8}, widgets.Text(m.Name, theme.SizeCallout, theme.T.Text, giode.FontWeightSemiBold)),
				widgets.Text(m.Vendor+" · "+tools.HumanBytes(m.Size)+" · "+m.License, theme.SizeCaption, theme.T.Muted)),
			action,
		),
		widgets.Text(tr(m.Description), theme.SizeFootnote, theme.T.Text2),
		giode.Row(widgets.St{Gap: 6, Wrap: giode.WrapWrap, Width: -1}, tags...),
	)
}

func vendorColor(v string) string {
	switch v {
	case "Alibaba":
		return "#a78bfa"
	case "Meta":
		return "#60a5fa"
	case "Google":
		return "#34d399"
	case "Microsoft":
		return "#38bdf8"
	case "Hugging Face":
		return "#f59e0b"
	}
	return "#8b93a7"
}
