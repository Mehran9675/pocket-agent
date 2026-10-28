package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/skills"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// newSkillRoute is the skill page's name for a skill being created.
const newSkillRoute = "+new"

const newSkillTemplate = "---\nname: my-skill\ndescription: What the skill does and when to use it, e.g. \"Use when the user asks for a weekly report.\"\n---\n\n" +
	"# My skill\n\nStep-by-step instructions the model follows when it uses this skill.\n"

func skillPageTitle(name string) string {
	if name == newSkillRoute {
		return "New skill"
	}
	return name
}

// skillEditorBody shows a skill's SKILL.md for editing, its files, and
// removal.
func (a *App) skillEditorBody(name string) widgets.El {
	creating := name == newSkillRoute
	var sk *skills.Skill
	if !creating {
		var err error
		if sk, err = a.skills.Get(name); err != nil {
			return widgets.Text("This skill was removed.", theme.SizeCallout, theme.T.Muted)
		}
	}
	key := "skill/src/" + name
	in, ok := a.inputs[key]
	if !ok {
		src := newSkillTemplate
		if !creating {
			src, _ = a.skills.Source(name)
		}
		in = a.input(key, "SKILL.md", src, func(i *input.Input) { i.Multiline(true) })
	}
	save := func() {
		prev := name
		if creating {
			prev = ""
		}
		s, err := a.skills.Save(prev, in.Text())
		if err != nil {
			a.toast(giode.ToastError, err.Error())
			return
		}
		a.dropInput(key)
		if s.Name != name {
			if !creating {
				a.setSkillEnabled(name, true)
			}
			a.setSkillEnabled(s.Name, true)
			a.back()
			a.navigate("settings/skill/" + s.Name)
		}
		a.toast(giode.ToastSuccess, "Saved "+s.Name)
	}
	kids := []widgets.El{
		widgets.Text("The frontmatter's name and description tell models what the skill is for; the rest is what they read when they use it.", theme.SizeFootnote, theme.T.Muted),
		giode.Box(widgets.St{Width: -1, MinHeight: 260}, in),
		widgets.Row(widgets.St{Gap: 10, Width: -1}, widgets.Spacer(),
			widgets.Button("skill/save", ifStr(creating, "Create skill", "Save"), "check", widgets.BtnPrimary, save)),
	}
	if sk != nil && len(sk.Files) > 0 {
		var rows []widgets.El
		for _, f := range sk.Files {
			rows = append(rows, widgets.ListRow("skill/f/"+f, "file-text", theme.T.Muted, f, "", nil, nil))
		}
		kids = append(kids, widgets.Col(widgets.St{Gap: 0, Width: -1}, widgets.SectionLabel("Files"), widgets.Group(rows...)))
	}
	if !creating {
		kids = append(kids, widgets.Button("skill/rm", "Remove skill", "trash-2", widgets.BtnDanger, func() {
			a.ask("Remove "+name+"?", "The skill and its files are deleted from this device.", "Remove", true, func() {
				if err := a.skills.Delete(name); err != nil {
					a.toast(giode.ToastError, err.Error())
					return
				}
				a.setSkillEnabled(name, true)
				a.dropInput(key)
				a.back()
			})
		}))
	}
	return widgets.Col(widgets.St{Gap: 14, Width: -1}, kids...)
}
