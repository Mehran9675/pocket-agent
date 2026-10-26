package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/local"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) hfSearch() widgets.El {
	in := a.input("hf/q", "Search Hugging Face GGUF models", "")
	in.Submit(func(q string) { a.runHFSearch(q) })
	var kids []widgets.El
	kids = append(kids, widgets.Row(widgets.St{Gap: 8, Width: -1},
		giode.Box(widgets.St{FlexGrow: 1}, in),
		widgets.Button("hf/go", "Search", "search", widgets.BtnSecondary, func() { a.runHFSearch(in.Text()) })))
	if a.hfBusy {
		kids = append(kids, widgets.Row(widgets.St{Gap: 8}, giode.Spinner(widgets.St{Width: 14, Height: 14, Color: theme.T.Primary}), widgets.Text("Searching…", theme.SizeFootnote, theme.T.Muted)))
	}
	if a.hfErr != "" {
		kids = append(kids, widgets.Text(a.hfErr, theme.SizeFootnote, theme.T.Danger))
	}
	var rows []widgets.El
	for _, r := range a.hfResults {
		r := r
		rows = append(rows, widgets.ListRow("hf/r/"+r.ID, "boxes", "#f59e0b", r.ID, fmt.Sprintf("%s downloads · %d likes", compactNum(r.Downloads), r.Likes), nil, func() { a.openRepo(r.ID) }))
	}
	if len(rows) > 0 {
		kids = append(kids, widgets.Group(rows...))
	}
	kids = append(kids, widgets.Text("Pick small instruction-tuned models (≤4B parameters, Q4_K_M) for phones. Split and vision-projector files aren't supported.", theme.SizeCaption, theme.T.Muted))
	return widgets.Col(widgets.St{Gap: 10, Width: -1}, kids...)
}

func (a *App) runHFSearch(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return
	}
	a.hfBusy, a.hfErr = true, ""
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		res, err := local.SearchHF(ctx, q)
		a.Post(func() {
			a.hfBusy = false
			if err != nil {
				a.hfErr = err.Error()
				return
			}
			a.hfResults = res
			if len(res) == 0 {
				a.hfErr = "No GGUF models found."
			}
		})
	}()
}
