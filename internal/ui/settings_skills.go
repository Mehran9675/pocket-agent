package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/skills"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) skillsBody() widgets.El {
	list := a.skills.List()
	var rows []widgets.El
	for _, s := range list {
		name := s.Name
		on := !slices.Contains(a.settings.DisabledSkills, name)
		rows = append(rows, widgets.ListRow("skill/"+name, "book-open", "#a78bfa", name, firstSentence(s.Description),
			widgets.Toggle("skill/t/"+name, on, func(v bool) { a.setSkillEnabled(name, v) }),
			func() { a.navigate("settings/skill/" + name) }))
	}
	urlIn := a.input("skills/url", "https://github.com/…/SKILL.md or a .zip link", "", func(i *input.Input) { i.Keyboard(giode.KeyboardURL) })
	kids := []widgets.El{
		widgets.Text("Skills teach models how to do specific tasks: a SKILL.md with a name, a description of when to use it and instructions, "+
			"plus any files it needs. Chats and agents see the enabled skills and load one when a task calls for it.", theme.SizeFootnote, theme.T.Muted),
	}
	if len(rows) > 0 {
		kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel(fmt.Sprintf("Installed · %d", len(list))), widgets.Group(rows...)))
	}
	kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel("Add a skill"), widgets.Card(widgets.St{Gap: 12},
		widgets.Row(widgets.St{Gap: 10, Width: -1, Wrap: giode.WrapWrap},
			widgets.Button("skills/file", "Install from file", "upload", widgets.BtnSecondary, a.installSkillFile),
			widgets.Button("skills/new", "Create a skill", "plus", widgets.BtnSecondary, func() { a.navigate("settings/skill/" + newSkillRoute) })),
		widgets.Text("A .zip with a SKILL.md inside, or a SKILL.md on its own.", theme.SizeFootnote, theme.T.Muted),
		urlIn,
		widgets.Button("skills/url/go", "Install from link", "download", widgets.BtnPrimary, func() {
			u := strings.TrimSpace(urlIn.Text())
			if !strings.HasPrefix(u, "http") {
				a.toast(giode.ToastWarning, "Enter a link to a SKILL.md or a .zip")
				return
			}
			a.toast(giode.ToastInfo, "Downloading skill…")
			go func() {
				s, err := a.skills.InstallURL(context.Background(), u)
				a.Post(func() {
					if err != nil {
						a.toast(giode.ToastError, err.Error())
						return
					}
					urlIn.SetText("")
					a.skillInstalled(s)
				})
			}()
		}))))
	return widgets.Col(widgets.St{Gap: 16, Width: -1}, kids...)
}

func (a *App) installSkillFile() {
	go func() {
		files, err := a.win.PickFiles(context.Background(), giode.PickOptions{})
		if errors.Is(err, giode.ErrCanceled) || len(files) == 0 {
			return
		}
		if err != nil {
			a.toast(giode.ToastError, err.Error())
			return
		}
		f := files[0]
		data, err := io.ReadAll(io.LimitReader(f, 8<<20+1))
		f.Close()
		var s *skills.Skill
		if err == nil {
			s, err = a.skills.Install(f.Name, data)
		}
		a.Post(func() {
			if err != nil {
				a.toast(giode.ToastError, err.Error())
				return
			}
			a.skillInstalled(s)
		})
	}()
}

// skillInstalled turns a newly installed skill on.
func (a *App) skillInstalled(s *skills.Skill) {
	a.setSkillEnabled(s.Name, true)
	a.toast(giode.ToastSuccess, "Installed "+s.Name)
}
