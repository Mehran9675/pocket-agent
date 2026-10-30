package ui

import (
	"context"
	"strings"
	"time"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/local"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) openRepo(repo string) {
	if _, ok := a.hfFiles[repo]; !ok {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			files, err := local.RepoFiles(ctx, repo)
			a.Post(func() {
				if err != nil {
					a.toast(giode.ToastError, err.Error())
					a.closeSheet()
					return
				}
				a.hfFiles[repo] = files
			})
		}()
	}
	a.openSheet("repo", repo, func() widgets.El {
		files, ok := a.hfFiles[repo]
		if !ok {
			return widgets.Row(widgets.St{Gap: 8, Padding: giode.UniformInset(12)}, giode.Spinner(widgets.St{Width: 16, Height: 16, Color: theme.T.Primary}), widgets.Text(tr("Loading files…"), theme.SizeFootnote, theme.T.Muted))
		}
		if len(files) == 0 {
			return widgets.Text(tr("This repository has no single-file GGUF models."), theme.SizeFootnote, theme.T.Muted)
		}
		var rows []widgets.El
		for _, f := range files {
			f := f
			name := lastPathElem(f.Path)
			fitNote := ""
			if a.ram > 0 && float64(f.Size)*1.4 > float64(a.ram)*0.6 {
				fitNote = tr(" · likely too large for this device")
			}
			rows = append(rows, widgets.ListRow("repo/f/"+f.Path, "download", "#34d399", name, tools.HumanBytes(f.Size)+fitNote, nil, func() {
				a.closeSheet()
				a.models.Start(local.Installed{File: name, Name: strings.TrimSuffix(name, ".gguf"), Repo: repo, Size: f.Size}, local.HFResolveURL(repo, f.Path))
				a.toast(giode.ToastInfo, tr("Downloading {name}", "name", name))
			}))
		}
		return widgets.Group(rows...)
	})
}
