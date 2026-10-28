// Package store holds Pocket Agent's data model and persists it as JSON
// files in the app's private data directory.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/local"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

// NewID returns a random identifier.
func NewID() string {
	var b [10]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// ModelRef names a model on a configured provider.
type ModelRef struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// IsZero reports whether no model is set.
func (r ModelRef) IsZero() bool { return r.Provider == "" || r.Model == "" }

// LocalProviderID is the provider ID of the on-device engine.
const LocalProviderID = "local"

// ProviderConfig is a provider the user set up.
type ProviderConfig struct {
	ID      string `json:"id"`
	Preset  string `json:"preset"`
	Name    string `json:"name"`
	BaseURL string `json:"baseURL,omitempty"`
	APIKey  string `json:"apiKey,omitempty"`
	Enabled bool   `json:"enabled"`
	// Models caches the provider's model list.
	Models []llm.ModelInfo `json:"models,omitempty"`
	// Pinned are model IDs shown first in the picker.
	Pinned []string `json:"pinned,omitempty"`
}

// Mode is how a conversation runs.
type Mode string

const (
	ModeChat     Mode = "chat"
	ModeAgent    Mode = "agent"
	ModeResearch Mode = "research"
)

// ResearchDepth selects how much a deep research run does.
type ResearchDepth string

const (
	DepthQuick    ResearchDepth = "quick"
	DepthStandard ResearchDepth = "standard"
	DepthDeep     ResearchDepth = "deep"
)

// Conversation is a chat thread.
type Conversation struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Model   ModelRef  `json:"model"`
	Mode    Mode      `json:"mode"`
	AgentID string    `json:"agentId,omitempty"`
	Pinned  bool      `json:"pinned,omitempty"`
	// Folder is the ID of the folder the conversation is in, "" for
	// none. Folders hold conversations only, one level deep.
	Folder string `json:"folder,omitempty"`
	// System overrides the default instructions for this thread.
	System string `json:"system,omitempty"`
	// WebSearch offers web tools in chat mode.
	WebSearch bool          `json:"webSearch,omitempty"`
	Depth     ResearchDepth `json:"depth,omitempty"`
	// Thinking sets how much reasoning models think in this chat.
	Thinking llm.Thinking `json:"thinking,omitempty"`
	Messages []*Message   `json:"messages"`
	// Version increments on every change, for render caches.
	Version int `json:"-"`
}

// Touch marks the conversation changed.
func (c *Conversation) Touch() {
	c.Updated = time.Now()
	c.Version++
}

// Preview is a one-line summary for lists.
func (c *Conversation) Preview() string {
	for i := len(c.Messages) - 1; i >= 0; i-- {
		m := c.Messages[i]
		t := m.Text
		if m.Role == llm.RoleAssistant {
			t = m.ReplyText()
		}
		if t = strings.TrimSpace(t); t != "" {
			t = strings.Join(strings.Fields(t), " ")
			if r := []rune(t); len(r) > 90 {
				t = string(r[:90]) + "…"
			}
			return t
		}
	}
	return ""
}

// BlockKind classifies parts of an assistant turn.
type BlockKind string

const (
	BlockText      BlockKind = "text"
	BlockReasoning BlockKind = "reasoning"
	BlockTool      BlockKind = "tool"
)

// ToolStatus is the state of a tool run.
type ToolStatus string

const (
	ToolPending  ToolStatus = "pending" // waiting for approval
	ToolRunning  ToolStatus = "running"
	ToolDone     ToolStatus = "done"
	ToolError    ToolStatus = "error"
	ToolDenied   ToolStatus = "denied"
	ToolCanceled ToolStatus = "canceled"
)

// ToolRun records one tool call.
type ToolRun struct {
	CallID   string         `json:"callId"`
	Name     string         `json:"name"`
	Args     string         `json:"args"`
	Result   string         `json:"result,omitempty"`
	IsError  bool           `json:"isError,omitempty"`
	Status   ToolStatus     `json:"status"`
	Started  time.Time      `json:"started"`
	Duration time.Duration  `json:"duration,omitempty"`
	Sources  []tools.Source `json:"sources,omitempty"`
	Files    []string       `json:"files,omitempty"`
}

// Block is one part of an assistant turn: text, visible thinking, or a
// tool call with its result.
type Block struct {
	Kind BlockKind `json:"kind"`
	Text string    `json:"text,omitempty"`
	Tool *ToolRun  `json:"tool,omitempty"`
}

// Attachment is a file attached to a user message. The bytes live in
// the attachments directory.
type Attachment struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Kind  string `json:"kind"` // image or document
	MIME  string `json:"mime"`
	Size  int64  `json:"size"`
	Pages int    `json:"pages,omitempty"`
	// Chars is the extracted text length of documents.
	Chars int `json:"chars,omitempty"`
}

// ResearchPhase is the stage of a deep research run.
type ResearchPhase string

const (
	PhasePlanning     ResearchPhase = "planning"
	PhaseSearching    ResearchPhase = "searching"
	PhaseReading      ResearchPhase = "reading"
	PhaseReflecting   ResearchPhase = "reflecting"
	PhaseWriting      ResearchPhase = "writing"
	PhaseDone         ResearchPhase = "done"
	PhaseFailed       ResearchPhase = "failed"
	PhaseCanceledPart ResearchPhase = "canceled"
)

// ResearchQuery is one line of inquiry.
type ResearchQuery struct {
	Question string `json:"question"`
	Query    string `json:"query"`
	Status   string `json:"status"` // pending, searching, reading, done, failed
	Found    int    `json:"found"`
	Read     int    `json:"read"`
	Round    int    `json:"round"`
}

// Research records a deep research run.
type Research struct {
	Phase   ResearchPhase   `json:"phase"`
	Goal    string          `json:"goal,omitempty"`
	Queries []ResearchQuery `json:"queries"`
	Sources []tools.Source  `json:"sources"`
	Notes   int             `json:"notes"`
	Round   int             `json:"round"`
	Rounds  int             `json:"rounds"`
	Started time.Time       `json:"started"`
	Elapsed time.Duration   `json:"elapsed,omitempty"`
}

// Message is one turn of a conversation.
type Message struct {
	ID      string    `json:"id"`
	Role    llm.Role  `json:"role"`
	Created time.Time `json:"created"`
	// Text is a user message's text.
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	// Blocks make up an assistant turn.
	Blocks  []Block   `json:"blocks,omitempty"`
	Model   string    `json:"model,omitempty"`
	Usage   llm.Usage `json:"usage,omitempty"`
	Error   string    `json:"error,omitempty"`
	Stopped bool      `json:"stopped,omitempty"`
	// Paused is set when an agent stopped at a limit (steps or tokens)
	// and can pick up where it left off.
	Paused   bool             `json:"paused,omitempty"`
	Plan     []tools.PlanItem `json:"plan,omitempty"`
	Research *Research        `json:"research,omitempty"`
	Duration time.Duration    `json:"duration,omitempty"`
	// Streaming is set while a reply is being generated.
	Streaming bool `json:"-"`
	Version   int  `json:"-"`
}

// ReplyText concatenates an assistant turn's text blocks.
func (m *Message) ReplyText() string {
	var b strings.Builder
	for _, bl := range m.Blocks {
		if bl.Kind == BlockText {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
				b.WriteString("\n\n")
			}
			b.WriteString(bl.Text)
		}
	}
	return b.String()
}

// Sources collects every source cited by tool runs in the turn.
func (m *Message) Sources() []tools.Source {
	seen := map[string]bool{}
	var out []tools.Source
	add := func(s tools.Source) {
		if s.URL == "" || seen[s.URL] {
			return
		}
		seen[s.URL] = true
		out = append(out, s)
	}
	if m.Research != nil {
		for _, s := range m.Research.Sources {
			add(s)
		}
	}
	for _, bl := range m.Blocks {
		if bl.Tool != nil && bl.Tool.Name == "fetch_url" {
			for _, s := range bl.Tool.Sources {
				add(s)
			}
		}
	}
	return out
}

// Files lists workspace files changed in the turn.
func (m *Message) Files() []string {
	seen := map[string]bool{}
	var out []string
	for _, bl := range m.Blocks {
		if bl.Tool == nil {
			continue
		}
		for _, f := range bl.Tool.Files {
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
	}
	return out
}

// Agent is a reusable agent configuration.
type Agent struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Icon         string   `json:"icon"`
	Color        string   `json:"color"`
	Description  string   `json:"description"`
	Instructions string   `json:"instructions"`
	Tools        []string `json:"tools"`
	// Model overrides the conversation's model when set.
	Model    ModelRef `json:"model,omitempty"`
	MaxSteps int      `json:"maxSteps"`
	// TokenBudget caps the tokens one task may use, in thousands; 0
	// uses DefaultTokenBudget.
	TokenBudget int `json:"tokenBudget,omitempty"`
	// AutoApprove runs tools that normally ask first without asking.
	AutoApprove bool     `json:"autoApprove,omitempty"`
	Starters    []string `json:"starters,omitempty"`
	Builtin     bool     `json:"builtin,omitempty"`
}

// Theme selects light or dark.
type Theme string

const (
	ThemeDark  Theme = "dark"
	ThemeLight Theme = "light"
)

// LocalSettings configure the on-device engine.
type LocalSettings struct {
	// ContextSize, Threads, GPULayers, Temperature and MaxTokens win over
	// the same fields in Load and Sampling.
	ContextSize int     `json:"contextSize"`
	Threads     int     `json:"threads"`
	GPULayers   int     `json:"gpuLayers"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
	// Load holds the advanced engine options: batches, flash attention,
	// KV cache types, loading.
	Load local.LoadOptions `json:"load"`
	// Sampling holds the samplers.
	Sampling local.GenParams `json:"sampling"`
}

// LoadOptions are the engine options to load a model with.
func (l LocalSettings) LoadOptions() local.LoadOptions {
	o := l.Load
	o.ContextSize, o.Threads, o.GPULayers = l.ContextSize, l.Threads, l.GPULayers
	return o
}

// GenParams are the sampling settings.
func (l LocalSettings) GenParams() local.GenParams {
	g := l.Sampling
	g.Temperature = float32(l.Temperature)
	if l.MaxTokens > 0 {
		g.MaxTokens = l.MaxTokens
	}
	return g
}

// Settings are the app preferences.
type Settings struct {
	Theme        Theme              `json:"theme"`
	Accent       string             `json:"accent"`
	DefaultModel ModelRef           `json:"defaultModel"`
	Providers    []*ProviderConfig  `json:"providers"`
	Search       tools.SearchConfig `json:"search"`
	// ChatTools are the tools offered in plain chat.
	ChatTools []string `json:"chatTools"`
	// AutoApprove lists tools allowed to run without asking.
	AutoApprove   map[string]bool `json:"autoApprove"`
	MemoryEnabled bool            `json:"memoryEnabled"`
	Instructions  string          `json:"instructions"`
	ShowReasoning bool            `json:"showReasoning"`
	// FrameTimes shows Giode's frame time counter (from the next launch).
	FrameTimes bool `json:"frameTimes,omitempty"`
	// Folders group conversations in the sidebar, in display order.
	Folders []*Folder `json:"folders,omitempty"`
	// Thinking is the level new chats start with: the last one chosen.
	Thinking   llm.Thinking      `json:"thinking,omitempty"`
	Local      LocalSettings     `json:"local"`
	MCPServers []tools.MCPServer `json:"mcpServers"`
	// DisabledSkills names installed skills turned off; new ones start on.
	DisabledSkills []string `json:"disabledSkills,omitempty"`
	// Onboarded is set once the welcome flow finished.
	Onboarded bool    `json:"onboarded"`
	FontScale float64 `json:"fontScale,omitempty"`
}

// DefaultSettings are used on first launch.
func DefaultSettings() *Settings {
	return &Settings{
		Theme:         ThemeDark,
		Accent:        "indigo",
		Search:        tools.SearchConfig{Engine: tools.SearchDuckDuckGo},
		ChatTools:     []string{"calculator", "current_datetime", "run_javascript", "read_attachment", "remember", "forget"},
		AutoApprove:   map[string]bool{},
		MemoryEnabled: true,
		ShowReasoning: true,
		Local:         LocalSettings{ContextSize: 4096, GPULayers: -1, Temperature: 0.7, MaxTokens: 2048, Sampling: local.DefaultGenParams()},
	}
}

// Provider returns the configured provider with id.
func (s *Settings) Provider(id string) *ProviderConfig {
	for _, p := range s.Providers {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// retiredAddresses are base URLs presets used to have.
var retiredAddresses = map[string]string{
	"together": "https://api.together.xyz/v1",
}

// dropPresetAddresses clears saved base URLs that are just a preset's
// address, current or retired, so connections follow the preset when
// it changes. Earlier versions saved the address even when unchanged.
func (s *Settings) dropPresetAddresses() {
	for _, p := range s.Providers {
		if p.BaseURL == "" {
			continue
		}
		base := strings.TrimRight(p.BaseURL, "/")
		pre, ok := llm.PresetByID(p.Preset)
		if (ok && base == strings.TrimRight(pre.BaseURL, "/")) || base == retiredAddresses[p.Preset] {
			p.BaseURL = ""
		}
	}
}

// DefaultTokenBudget is the tokens, in thousands, a task may use before
// pausing.
const DefaultTokenBudget = 500

// Budget is the agent's token budget in tokens.
func (a *Agent) Budget() int {
	if a == nil || a.TokenBudget <= 0 {
		return DefaultTokenBudget * 1000
	}
	return a.TokenBudget * 1000
}

// Folder groups conversations in the sidebar.
type Folder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Collapsed hides its conversations in the sidebar.
	Collapsed bool `json:"collapsed,omitempty"`
}

// Folder returns the folder with id, or nil.
func (s *Settings) Folder(id string) *Folder {
	for _, f := range s.Folders {
		if f.ID == id {
			return f
		}
	}
	return nil
}
