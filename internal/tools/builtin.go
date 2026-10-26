package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// Builtins returns every built-in tool, in display order.
func Builtins() []*Tool {
	return []*Tool{
		webSearchTool(),
		fetchTool(),
		calculatorTool(),
		dateTimeTool(),
		javascriptTool(),
		readAttachmentTool(),
		listFilesTool(),
		readFileTool(),
		writeFileTool(),
		editFileTool(),
		deleteFileTool(),
		planTool(),
		rememberTool(),
		forgetTool(),
	}
}

// Builtin returns the built-in tool named name.
func Builtin(name string) (*Tool, bool) {
	for _, t := range Builtins() {
		if t.Name == name {
			return t, true
		}
	}
	return nil, false
}

func calculatorTool() *Tool {
	return &Tool{
		Name:  "calculator",
		Title: "Calculator",
		Icon:  "calculator",
		Group: "Compute",
		Description: "Evaluate an arithmetic expression exactly instead of computing in your head. " +
			"Supports + - * / % ^ !, parentheses, pi, e and sqrt, abs, ln, log, log2, exp, sin, cos, tan, asin, acos, atan, floor, ceil, round, min, max, pow.",
		Schema: object(map[string]any{"expression": str("The expression, e.g. (1.07^10 - 1) * 2500")}, "expression"),
		Run: func(_ context.Context, _ *Env, raw json.RawMessage) Result {
			var a struct {
				Expression string `json:"expression"`
			}
			_ = json.Unmarshal(raw, &a)
			v, err := Evaluate(a.Expression)
			if err != nil {
				return Errorf("can't evaluate %q: %v", a.Expression, err)
			}
			return Result{Text: a.Expression + " = " + FormatNumber(v)}
		},
	}
}

func dateTimeTool() *Tool {
	return &Tool{
		Name:        "current_datetime",
		Title:       "Date & time",
		Icon:        "clock",
		Group:       "Compute",
		Description: "Get the current date, time, weekday and time zone of the user's device, optionally converted to another IANA time zone.",
		Schema:      object(map[string]any{"timezone": str("Optional IANA time zone, e.g. Europe/Paris.")}),
		Run: func(_ context.Context, _ *Env, raw json.RawMessage) Result {
			var a struct {
				Timezone string `json:"timezone"`
			}
			_ = json.Unmarshal(raw, &a)
			now := time.Now()
			if a.Timezone != "" {
				loc, err := time.LoadLocation(a.Timezone)
				if err != nil {
					return Errorf("unknown time zone %q", a.Timezone)
				}
				now = now.In(loc)
			}
			zone, off := now.Zone()
			return Result{Text: fmt.Sprintf("%s (%s, UTC%+.1f), ISO %s", now.Format("Monday, 2 January 2006 15:04:05"), zone, float64(off)/3600, now.Format(time.RFC3339))}
		},
	}
}

func javascriptTool() *Tool {
	return &Tool{
		Name:     "run_javascript",
		Title:    "Run code",
		Icon:     "square-terminal",
		Group:    "Compute",
		Approval: false,
		Description: "Run JavaScript (ES2020, no network or DOM) in a sandbox and return console output and the value of the last expression. " +
			"Use it for data processing, statistics, parsing, simulations and precise computation. " +
			"In agent mode, readFile(path) and writeFile(path, text) access the workspace. Execution is limited to 10 seconds.",
		Schema: object(map[string]any{"code": str("The JavaScript source to run.")}, "code"),
		Run: func(ctx context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Code string `json:"code"`
			}
			_ = json.Unmarshal(raw, &a)
			return runJS(ctx, env, a.Code)
		},
	}
}

func runJS(ctx context.Context, env *Env, code string) Result {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	var out strings.Builder
	logf := func(prefix string) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			parts := make([]string, len(call.Arguments))
			for i, arg := range call.Arguments {
				parts[i] = jsString(vm, arg)
			}
			if out.Len() < 64<<10 {
				out.WriteString(prefix + strings.Join(parts, " ") + "\n")
			}
			return goja.Undefined()
		}
	}
	console := vm.NewObject()
	_ = console.Set("log", logf(""))
	_ = console.Set("info", logf(""))
	_ = console.Set("warn", logf("warn: "))
	_ = console.Set("error", logf("error: "))
	_ = vm.Set("console", console)
	var changed []string
	if env != nil && env.Workspace != "" {
		_ = vm.Set("readFile", func(p string) (string, error) {
			full, err := workspacePath(env, p)
			if err != nil {
				return "", err
			}
			b, err := os.ReadFile(full)
			return string(b), err
		})
		_ = vm.Set("writeFile", func(p, text string) error {
			full, err := workspacePath(env, p)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return err
			}
			changed = append(changed, filepath.ToSlash(p))
			return os.WriteFile(full, []byte(text), 0o644)
		})
	}
	timer := time.AfterFunc(10*time.Second, func() { vm.Interrupt("time limit of 10 seconds exceeded") })
	defer timer.Stop()
	stop := context.AfterFunc(ctx, func() { vm.Interrupt("canceled") })
	defer stop()
	v, err := vm.RunString(code)
	text := out.String()
	if err != nil {
		var ex *goja.Exception
		msg := err.Error()
		if errors.As(err, &ex) {
			msg = ex.Error()
		}
		return Result{Text: strings.TrimSpace(text + "\nError: " + msg), IsError: true, Files: changed}
	}
	if v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		text += "=> " + jsString(vm, v)
	}
	if strings.TrimSpace(text) == "" {
		text = "(no output - use console.log to print results)"
	}
	return Result{Text: truncate(text, 20000), Files: changed}
}

func jsString(vm *goja.Runtime, v goja.Value) string {
	if v == nil {
		return "undefined"
	}
	switch v.ExportType() {
	case nil:
		return v.String()
	}
	if obj, ok := v.(*goja.Object); ok && obj.ClassName() != "Function" && obj.ClassName() != "Error" {
		if stringify, ok := goja.AssertFunction(vm.Get("JSON").ToObject(vm).Get("stringify")); ok {
			if s, err := stringify(goja.Undefined(), v, goja.Undefined(), vm.ToValue(2)); err == nil && !goja.IsUndefined(s) {
				return s.String()
			}
		}
	}
	return v.String()
}

// workspacePath resolves p inside the workspace, refusing escapes.
func workspacePath(env *Env, p string) (string, error) {
	if env == nil || env.Workspace == "" {
		return "", errors.New("no workspace - file tools work in agent mode")
	}
	clean := filepath.Clean("/" + filepath.FromSlash(strings.TrimSpace(p)))
	if clean == "/" || clean == string(filepath.Separator) {
		return env.Workspace, nil
	}
	return filepath.Join(env.Workspace, clean), nil
}

func listFilesTool() *Tool {
	return &Tool{
		Name:        "list_files",
		Title:       "List files",
		Icon:        "folder",
		Group:       "Workspace",
		Description: "List the files in the agent workspace (recursively), with sizes.",
		Schema:      object(map[string]any{"path": str("Optional sub-directory, default the workspace root.")}),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Path string `json:"path"`
			}
			_ = json.Unmarshal(raw, &a)
			root, err := workspacePath(env, a.Path)
			if err != nil {
				return Errorf("%v", err)
			}
			var lines []string
			_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
				if err != nil || p == root {
					return nil
				}
				rel, _ := filepath.Rel(env.Workspace, p)
				if d.IsDir() {
					lines = append(lines, filepath.ToSlash(rel)+"/")
					return nil
				}
				info, _ := d.Info()
				lines = append(lines, fmt.Sprintf("%s (%s)", filepath.ToSlash(rel), HumanBytes(info.Size())))
				return nil
			})
			if len(lines) == 0 {
				return Result{Text: "The workspace is empty."}
			}
			sort.Strings(lines)
			return Result{Text: strings.Join(lines, "\n")}
		},
	}
}

func readFileTool() *Tool {
	return &Tool{
		Name:        "read_file",
		Title:       "Read file",
		Icon:        "file-text",
		Group:       "Workspace",
		Description: "Read a text file from the agent workspace. Long files are paged with offset (in characters).",
		Schema:      object(map[string]any{"path": str("Path relative to the workspace."), "offset": integer("Character offset to start at.")}, "path"),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Path   string `json:"path"`
				Offset int    `json:"offset"`
			}
			_ = json.Unmarshal(raw, &a)
			full, err := workspacePath(env, a.Path)
			if err != nil {
				return Errorf("%v", err)
			}
			b, err := os.ReadFile(full)
			if err != nil {
				return Errorf("can't read %s: %v", a.Path, err)
			}
			text := []rune(string(b))
			if a.Offset > 0 && a.Offset < len(text) {
				text = text[a.Offset:]
			}
			return Result{Text: truncate(string(text), 16000)}
		},
	}
}

func writeFileTool() *Tool {
	return &Tool{
		Name:        "write_file",
		Title:       "Write file",
		Icon:        "pen-line",
		Group:       "Workspace",
		Description: "Create or overwrite a file in the agent workspace - reports, code, data, notes. The user can open, share and save workspace files.",
		Schema:      object(map[string]any{"path": str("Path relative to the workspace, e.g. report.md or src/main.py."), "content": str("The full file content.")}, "path", "content"),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			_ = json.Unmarshal(raw, &a)
			full, err := workspacePath(env, a.Path)
			if err != nil {
				return Errorf("%v", err)
			}
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return Errorf("%v", err)
			}
			if err := os.WriteFile(full, []byte(a.Content), 0o644); err != nil {
				return Errorf("can't write %s: %v", a.Path, err)
			}
			return Result{Text: fmt.Sprintf("Wrote %s (%s).", a.Path, HumanBytes(int64(len(a.Content)))), Files: []string{a.Path}}
		},
	}
}

func editFileTool() *Tool {
	return &Tool{
		Name:        "edit_file",
		Title:       "Edit file",
		Icon:        "pencil",
		Group:       "Workspace",
		Description: "Replace an exact snippet in a workspace file. old_text must occur exactly once.",
		Schema: object(map[string]any{
			"path":     str("Path relative to the workspace."),
			"old_text": str("The exact text to replace."),
			"new_text": str("The replacement."),
		}, "path", "old_text", "new_text"),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Path    string `json:"path"`
				OldText string `json:"old_text"`
				NewText string `json:"new_text"`
			}
			_ = json.Unmarshal(raw, &a)
			full, err := workspacePath(env, a.Path)
			if err != nil {
				return Errorf("%v", err)
			}
			b, err := os.ReadFile(full)
			if err != nil {
				return Errorf("can't read %s: %v", a.Path, err)
			}
			s := string(b)
			switch n := strings.Count(s, a.OldText); {
			case a.OldText == "" || n == 0:
				return Errorf("old_text not found in %s", a.Path)
			case n > 1:
				return Errorf("old_text occurs %d times in %s; include more context", n, a.Path)
			}
			s = strings.Replace(s, a.OldText, a.NewText, 1)
			if err := os.WriteFile(full, []byte(s), 0o644); err != nil {
				return Errorf("%v", err)
			}
			return Result{Text: "Edited " + a.Path + ".", Files: []string{a.Path}}
		},
	}
}

func deleteFileTool() *Tool {
	return &Tool{
		Name:        "delete_file",
		Title:       "Delete file",
		Icon:        "trash-2",
		Group:       "Workspace",
		Approval:    true,
		Description: "Delete a file or directory from the agent workspace.",
		Schema:      object(map[string]any{"path": str("Path relative to the workspace.")}, "path"),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Path string `json:"path"`
			}
			_ = json.Unmarshal(raw, &a)
			full, err := workspacePath(env, a.Path)
			if err != nil || full == env.Workspace {
				return Errorf("refusing to delete %q", a.Path)
			}
			if err := os.RemoveAll(full); err != nil {
				return Errorf("%v", err)
			}
			return Result{Text: "Deleted " + a.Path + ".", Files: []string{a.Path}}
		},
	}
}

func planTool() *Tool {
	return &Tool{
		Name:  "update_plan",
		Title: "Plan",
		Icon:  "list-checks",
		Group: "Agent",
		Description: "Create or update your step-by-step plan for a multi-step task; the user sees it as a live checklist. " +
			"Send the whole list each time with each step's status: pending, in_progress or completed. Keep one step in_progress.",
		Schema: object(map[string]any{
			"steps": map[string]any{
				"type":        "array",
				"description": "The full plan.",
				"items": object(map[string]any{
					"step":   str("Short imperative description."),
					"status": map[string]any{"type": "string", "enum": []string{"pending", "in_progress", "completed"}},
				}, "step", "status"),
			},
		}, "steps"),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Steps []PlanItem `json:"steps"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return Errorf("steps must be a list of {step, status}")
			}
			if env != nil && env.OnPlan != nil {
				env.OnPlan(a.Steps)
			}
			done := 0
			for _, s := range a.Steps {
				if s.Status == "completed" {
					done++
				}
			}
			return Result{Text: fmt.Sprintf("Plan updated (%d/%d done).", done, len(a.Steps))}
		},
	}
}

func rememberTool() *Tool {
	return &Tool{
		Name:  "remember",
		Title: "Remember",
		Icon:  "brain",
		Group: "Memory",
		Description: "Save a durable fact about the user (preferences, background, ongoing projects) to long-term memory, " +
			"available in future conversations. Only save what the user would want remembered.",
		Schema: object(map[string]any{"fact": str("One concise fact, written in the third person.")}, "fact"),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Fact string `json:"fact"`
			}
			_ = json.Unmarshal(raw, &a)
			if env == nil || env.Memory == nil {
				return Errorf("memory is turned off")
			}
			if err := env.Memory.Remember(a.Fact); err != nil {
				return Errorf("%v", err)
			}
			return Result{Text: "Saved to memory."}
		},
	}
}

func forgetTool() *Tool {
	return &Tool{
		Name:        "forget",
		Title:       "Forget",
		Icon:        "brain",
		Group:       "Memory",
		Description: "Remove a fact from long-term memory when the user asks you to forget it or it's no longer true.",
		Schema:      object(map[string]any{"fact": str("The fact to remove, as stored.")}, "fact"),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Fact string `json:"fact"`
			}
			_ = json.Unmarshal(raw, &a)
			if env == nil || env.Memory == nil {
				return Errorf("memory is turned off")
			}
			if !env.Memory.Forget(a.Fact) {
				return Errorf("no such memory")
			}
			return Result{Text: "Forgotten."}
		},
	}
}

func readAttachmentTool() *Tool {
	return &Tool{
		Name:        "read_attachment",
		Title:       "Read attachment",
		Icon:        "paperclip",
		Group:       "Files",
		Description: "Read more of a document the user attached to this conversation (the prompt only includes the beginning of long ones). Pages through the text with offset.",
		Schema:      object(map[string]any{"name": str("The attachment's file name."), "offset": integer("Character offset to start at.")}, "name"),
		Run: func(_ context.Context, env *Env, raw json.RawMessage) Result {
			var a struct {
				Name   string `json:"name"`
				Offset int    `json:"offset"`
			}
			_ = json.Unmarshal(raw, &a)
			if env == nil || env.Attachments == nil {
				return Errorf("no attachments")
			}
			var names []string
			for _, att := range env.Attachments() {
				names = append(names, att.Name)
				if !strings.EqualFold(att.Name, a.Name) {
					continue
				}
				text := []rune(att.Text)
				if a.Offset > 0 && a.Offset < len(text) {
					text = text[a.Offset:]
				}
				const page = 16000
				if len(text) > page {
					return Result{Text: string(text[:page]) + fmt.Sprintf("\n\n[... %d more characters; continue with offset=%d]", len(text)-page, a.Offset+page)}
				}
				return Result{Text: string(text)}
			}
			return Errorf("no attachment named %q; attachments: %s", a.Name, strings.Join(names, ", "))
		},
	}
}

// HumanBytes formats a size.
func HumanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}
