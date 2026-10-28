package tools

import (
	"context"
	"encoding/json"
)

// SkillTools are offered whenever the user has skills enabled.
func SkillTools() []*Tool {
	return []*Tool{useSkillTool(), readSkillFileTool()}
}

func useSkillTool() *Tool {
	return &Tool{
		Name:  "use_skill",
		Title: "Use skill",
		Icon:  "book-open",
		Group: "Skills",
		Description: "Load the full instructions of one of the user's installed skills. Call it before doing a task " +
			"that a listed skill covers, then follow what it says.",
		Schema: object(map[string]any{"name": str("The skill's name, as listed.")}, "name"),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Name string `json:"name"`
			}
			_ = json.Unmarshal(raw, &a)
			if env == nil || env.Skills == nil {
				return Errorf("no skills are enabled")
			}
			text, err := env.Skills.Load(a.Name)
			if err != nil {
				return Errorf("%v", err)
			}
			return Result{Text: text}
		},
	}
}

func readSkillFileTool() *Tool {
	return &Tool{
		Name:        "read_skill_file",
		Title:       "Read skill file",
		Icon:        "file-text",
		Group:       "Skills",
		Description: "Read a file bundled with a skill, such as a reference or template its instructions point to.",
		Schema: object(map[string]any{
			"name": str("The skill's name."),
			"path": str("The file's path inside the skill, e.g. reference/forms.md."),
		}, "name", "path"),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Name string `json:"name"`
				Path string `json:"path"`
			}
			_ = json.Unmarshal(raw, &a)
			if env == nil || env.Skills == nil {
				return Errorf("no skills are enabled")
			}
			text, err := env.Skills.File(a.Name, a.Path)
			if err != nil {
				return Errorf("%v", err)
			}
			return Result{Text: text}
		},
	}
}
