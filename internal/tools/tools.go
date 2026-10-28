// Package tools holds the tools models can call: web search and fetch,
// a calculator, a JavaScript sandbox, the agent workspace, the plan
// checklist, long-term memory, attachment reading - and tools served
// by MCP servers, adapted in mcp.go.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mehran9675/pocket-agent/internal/llm"
)

// Source is a web page a tool consulted, shown as a citation.
type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
}

// PlanItem is one step of an agent's plan.
type PlanItem struct {
	Step   string `json:"step"`
	Status string `json:"status"` // pending, in_progress, completed
}

// Result is a tool's output.
type Result struct {
	// Text goes back to the model.
	Text    string
	IsError bool
	// Sources are pages cited by the result.
	Sources []Source
	// Files lists workspace paths the tool created or changed.
	Files []string
}

// Errorf returns an error result the model sees.
func Errorf(format string, a ...any) Result {
	return Result{Text: fmt.Sprintf(format, a...), IsError: true}
}

// Env is what tools may touch. Any field may be nil/empty, in which case
// the tools depending on it report they're unavailable.
type Env struct {
	// Workspace is the agent's sandbox directory.
	Workspace string
	// Search configures web search.
	Search SearchConfig
	// Attachments returns the documents attached to the conversation.
	Attachments func() []Attachment
	// Memory is the long-term memory store.
	Memory Memory
	// OnPlan receives plan updates.
	OnPlan func([]PlanItem)
	// Skills serves the enabled skills.
	Skills Skills
}

// Skills gives models the skills the user installed.
type Skills interface {
	// Load returns a skill's instructions and lists its files.
	Load(name string) (string, error)
	// File returns one of a skill's files.
	File(name, path string) (string, error)
}

// Attachment is a readable document attached to the conversation.
type Attachment struct {
	Name string
	Text string
}

// Memory persists facts about the user across conversations.
type Memory interface {
	Remember(fact string) error
	Forget(fact string) bool
	All() []string
}

// Tool is a callable tool.
type Tool struct {
	Name        string
	Title       string
	Description string
	// Icon names an icon in the app's icon registry.
	Icon   string
	Schema map[string]any
	// Approval marks tools that change things outside the app or run
	// code; they ask the user first unless allowed in settings.
	Approval bool
	// Group buckets tools in settings.
	Group string
	Run   func(ctx context.Context, env *Env, args json.RawMessage) Result
}

// Spec converts t for a model request.
func (t *Tool) Spec() llm.ToolSpec {
	return llm.ToolSpec{Name: t.Name, Description: t.Description, Schema: t.Schema}
}

// Set is a group of tools offered to a model.
type Set struct {
	byName map[string]*Tool
	order  []string
}

// NewSet builds a set from tools.
func NewSet(tools ...*Tool) *Set {
	s := &Set{byName: map[string]*Tool{}}
	for _, t := range tools {
		s.Add(t)
	}
	return s
}

// Add adds or replaces a tool.
func (s *Set) Add(t *Tool) {
	if _, ok := s.byName[t.Name]; !ok {
		s.order = append(s.order, t.Name)
	}
	s.byName[t.Name] = t
}

// Get returns the tool named name.
func (s *Set) Get(name string) (*Tool, bool) {
	if s == nil {
		return nil, false
	}
	t, ok := s.byName[name]
	return t, ok
}

// Len counts the tools.
func (s *Set) Len() int {
	if s == nil {
		return 0
	}
	return len(s.order)
}

// Tools lists the tools in insertion order.
func (s *Set) Tools() []*Tool {
	if s == nil {
		return nil
	}
	out := make([]*Tool, 0, len(s.order))
	for _, n := range s.order {
		out = append(out, s.byName[n])
	}
	return out
}

// Specs lists the tools for a model request.
func (s *Set) Specs() []llm.ToolSpec {
	var out []llm.ToolSpec
	for _, t := range s.Tools() {
		out = append(out, t.Spec())
	}
	return out
}

// Run executes the named tool, validating its JSON arguments first.
func (s *Set) Run(ctx context.Context, env *Env, call llm.ToolCall) (res Result) {
	t, ok := s.Get(call.Name)
	if !ok {
		return Errorf("unknown tool %q", call.Name)
	}
	args := call.Args
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	if !json.Valid(args) {
		return Errorf("INVALID_JSON: the arguments for %s were not valid JSON; call the tool again with a complete JSON object", call.Name)
	}
	if err := checkRequired(t.Schema, args); err != nil {
		return Errorf("%s: %v", call.Name, err)
	}
	defer func() {
		if r := recover(); r != nil {
			res = Errorf("%s crashed: %v", call.Name, r)
		}
	}()
	return t.Run(ctx, env, args)
}

// checkRequired verifies the arguments object has the schema's required
// keys - the cheap part of validation that catches truncated input.
func checkRequired(schema map[string]any, args json.RawMessage) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(args, &obj); err != nil {
		return fmt.Errorf("arguments must be a JSON object")
	}
	var missing []string
	switch req := schema["required"].(type) {
	case []string:
		for _, k := range req {
			if _, ok := obj[k]; !ok {
				missing = append(missing, k)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing required argument(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

// object builds a JSON Schema object from properties and required keys.
func object(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func integer(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

// truncate shortens s to n runes, noting the cut.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + fmt.Sprintf("\n\n[... truncated, %d more characters]", len(r)-n)
}
