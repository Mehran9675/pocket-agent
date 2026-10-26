package store

import (
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
