package ui

import (
	"time"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) researchCard(m *store.Message) widgets.El {
	r := m.Research
	key := m.ID + "/research"
	finished := r.Phase == store.PhaseDone || r.Phase == store.PhaseFailed || r.Phase == store.PhaseCanceledPart
	open := a.expanded[key] || !finished
	phases := []struct {
		p     store.ResearchPhase
		label string
	}{
		{store.PhasePlanning, tr("Plan")}, {store.PhaseSearching, tr("Search")}, {store.PhaseReading, tr("Read")},
		{store.PhaseReflecting, tr("Reflect")}, {store.PhaseWriting, tr("Write")},
	}
	order := map[store.ResearchPhase]int{store.PhasePlanning: 0, store.PhaseSearching: 1, store.PhaseReading: 2, store.PhaseReflecting: 3, store.PhaseWriting: 4, store.PhaseDone: 5}
	cur := order[r.Phase]
	var steps []widgets.El
	for i, ph := range phases {
		color, fg := theme.T.Surface2, theme.T.Faint
		switch {
		case i < cur || r.Phase == store.PhaseDone:
			color, fg = theme.T.PrimarySoft, theme.T.PrimaryText
		case i == cur && !finished:
			color, fg = theme.T.Primary, theme.T.OnPrimary
		}
		steps = append(steps, giode.Box(widgets.St{FlexGrow: 1, Padding: giode.SymmetricInset(0, 5), BorderRadius: 8, Background: color, Justify: giode.JustifyCenter},
			widgets.Text(ph.label, theme.SizeCaption, fg, giode.FontWeightMedium)))
	}
	status := map[store.ResearchPhase]string{
		store.PhasePlanning: tr("Planning the research"), store.PhaseSearching: tr("Searching the web"),
		store.PhaseReading: tr("Reading sources"), store.PhaseReflecting: tr("Looking for gaps"),
		store.PhaseWriting: tr("Writing the report"), store.PhaseDone: tr("Research complete"),
		store.PhaseFailed: tr("Research failed"), store.PhaseCanceledPart: tr("Research stopped"),
	}[r.Phase]
	elapsed := r.Elapsed
	if !finished && !r.Started.IsZero() {
		elapsed = time.Since(r.Started)
	}
	summary := trn("{n} sources", len(r.Sources)) + " · " + trn("{n} notes", r.Notes) + " · " + fmtDuration(elapsed.Round(time.Second))
	head := widgets.Tap(key, tr("Research progress"), func(s giode.PressState) widgets.El {
		var lead widgets.El = giode.Spinner(widgets.St{Width: 18, Height: 18, Color: "#38bdf8"})
		if finished {
			lead = widgets.IconTile("telescope", "#38bdf8", 30)
		}
		chev := "chevron-down"
		if open {
			chev = "chevron-up"
		}
		return widgets.Row(widgets.St{Gap: 10, Width: -1},
			lead,
			widgets.Col(widgets.St{Gap: 2, FlexGrow: 1},
				widgets.Text(status, theme.SizeCallout, theme.T.Text, giode.FontWeightSemiBold),
				widgets.Line(summary, theme.SizeFootnote, theme.T.Muted)),
			widgets.When(finished, func() widgets.El { return widgets.Icon(chev, 16, theme.T.Faint) }))
	}, func() { a.expanded[key] = !a.expanded[key] })
	kids := []widgets.El{head}
	if open {
		kids = append(kids, widgets.Row(widgets.St{Gap: 4, Width: -1}, steps...))
		if r.Goal != "" {
			kids = append(kids, widgets.Text(r.Goal, theme.SizeFootnote, theme.T.Text2))
		}
		var qs []widgets.El
		for _, q := range r.Queries {
			var icon widgets.El
			switch q.Status {
			case "done":
				icon = widgets.Icon("check", 14, theme.T.Success)
			case "failed":
				icon = widgets.Icon("circle-x", 14, theme.T.Danger)
			case "pending":
				icon = widgets.Icon("circle", 14, theme.T.Faint)
			default:
				if finished {
					icon = widgets.Icon("circle", 14, theme.T.Faint)
				} else {
					icon = giode.Spinner(widgets.St{Width: 13, Height: 13, Color: "#38bdf8"})
				}
			}
			detail := ""
			if q.Found > 0 {
				detail = trn("{n} read", q.Read)
			}
			qs = append(qs, widgets.Row(widgets.St{Gap: 10, Width: -1},
				icon,
				giode.Box(widgets.St{FlexGrow: 1}, widgets.Line(q.Query, theme.SizeFootnote, theme.T.Text2)),
				widgets.Text(detail, theme.SizeCaption, theme.T.Faint)))
		}
		if len(qs) > 0 {
			kids = append(kids, widgets.Col(widgets.St{Gap: 7, Width: -1, Padding: giode.Inset{Top: 2}}, qs...))
		}
	}
	return widgets.Card(widgets.St{Gap: 12, BorderColor: theme.Tint("#38bdf8", 0.25)}, kids...)
}
