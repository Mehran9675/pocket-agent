package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mehran9675/pocket-agent/internal/skills"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

// enabledSkills lists the installed skills that aren't turned off.
func (a *App) enabledSkills() []*skills.Skill {
	var out []*skills.Skill
	for _, s := range a.skills.List() {
		if !slices.Contains(a.settings.DisabledSkills, s.Name) {
			out = append(out, s)
		}
	}
	return out
}

// skillIndex is what the system prompt lists: the enabled skills, when
// the model can call use_skill to load them.
func (a *App) skillIndex(set *tools.Set, canCall bool) [][2]string {
	if _, ok := set.Get("use_skill"); !ok || !canCall {
		return nil
	}
	var out [][2]string
	for _, s := range a.enabledSkills() {
		out = append(out, [2]string{s.Name, s.Description})
	}
	return out
}

func (a *App) setSkillEnabled(name string, on bool) {
	a.settings.DisabledSkills = slices.DeleteFunc(a.settings.DisabledSkills, func(n string) bool { return n == name })
	if !on {
		a.settings.DisabledSkills = append(a.settings.DisabledSkills, name)
	}
	a.saveSettings()
}

// skillServer serves a turn's enabled skills to the skill tools.
type skillServer struct {
	lib     *skills.Library
	enabled []*skills.Skill
}

func (s skillServer) find(name string) (*skills.Skill, error) {
	name = strings.TrimSpace(name)
	for _, sk := range s.enabled {
		if sk.Name == name {
			return sk, nil
		}
	}
	return nil, fmt.Errorf("no enabled skill named %q", name)
}

func (s skillServer) Load(name string) (string, error) {
	sk, err := s.find(name)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(sk.Body)
	if len(sk.Files) > 0 {
		text += "\n\n[Files in this skill, readable with read_skill_file: " + strings.Join(sk.Files, ", ") + "]"
	}
	return text, nil
}

func (s skillServer) File(name, path string) (string, error) {
	if _, err := s.find(name); err != nil {
		return "", err
	}
	return s.lib.ReadFile(name, path)
}
