package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mehran9675/pocket-agent/internal/llm"
)

func TestConversationRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := &Conversation{ID: NewID(), Title: "T", Updated: time.Now(), Mode: ModeAgent, Messages: []*Message{
		{ID: "u", Role: llm.RoleUser, Text: "hi"},
		{ID: "a", Role: llm.RoleAssistant, Blocks: []Block{{Kind: BlockTool, Tool: &ToolRun{CallID: "x", Name: "n", Status: ToolRunning}}}},
	}}
	s.SaveConversation(c)
	st := s.LoadSettings()
	st.Instructions = "be brief"
	s.SaveSettings(st)
	if err := s.Remember("likes tea"); err != nil {
		t.Fatal(err)
	}
	s.Flush()

	s2, _ := Open(dir)
	cs := s2.LoadConversations()
	if len(cs) != 1 || cs[0].Title != "T" || cs[0].Messages[1].Blocks[0].Tool.Status != ToolCanceled {
		t.Fatalf("loaded %+v", cs)
	}
	if s2.LoadSettings().Instructions != "be brief" || len(s2.All()) != 1 {
		t.Fatal("settings or memory not persisted")
	}
	if len(s2.LoadAgents()) != len(BuiltinAgents()) {
		t.Fatal("agents")
	}
}

type memSecrets map[string][]byte

func (m memSecrets) Set(name string, v []byte) error { m[name] = v; return nil }
func (m memSecrets) Get(name string) ([]byte, error) {
	if v, ok := m[name]; ok {
		return v, nil
	}
	return nil, os.ErrNotExist
}

func TestKeysInSecrets(t *testing.T) {
	dir := t.TempDir()
	sec := memSecrets{}
	s, _ := Open(dir)
	if !s.UseSecrets(sec) {
		t.Fatal("probe failed")
	}
	st := s.LoadSettings()
	st.Providers = append(st.Providers, &ProviderConfig{ID: "p1", APIKey: "sk-123"})
	st.Search.APIKey = "brave"
	s.SaveSettings(st)
	s.Flush()
	raw, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	if strings.Contains(string(raw), "sk-123") || strings.Contains(string(raw), "brave") {
		t.Fatalf("key written to file: %s", raw)
	}
	s2, _ := Open(dir)
	s2.UseSecrets(sec)
	st2 := s2.LoadSettings()
	if st2.Providers[len(st2.Providers)-1].APIKey != "sk-123" || st2.Search.APIKey != "brave" {
		t.Fatal("keys not restored")
	}
}

func TestDropPresetAddresses(t *testing.T) {
	st := DefaultSettings()
	st.Providers = []*ProviderConfig{
		{ID: "together", Preset: "together", BaseURL: "https://api.together.xyz/v1"},
		{ID: "groq", Preset: "groq", BaseURL: "https://api.groq.com/openai/v1/"},
		{ID: "ollama", Preset: "ollama", BaseURL: "http://10.0.0.5:11434/v1"},
	}
	st.dropPresetAddresses()
	if st.Providers[0].BaseURL != "" || st.Providers[1].BaseURL != "" || st.Providers[2].BaseURL == "" {
		t.Fatalf("addresses: %q %q %q", st.Providers[0].BaseURL, st.Providers[1].BaseURL, st.Providers[2].BaseURL)
	}
}
