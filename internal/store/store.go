package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store persists the app's data under one directory:
//
//	settings.json, agents.json, memory.json
//	conversations/<id>.json
//	attachments/<id>, attachments/<id>.txt
//	workspaces/<conversation id>/...
//	models/*.gguf
type Store struct {
	dir string

	writeMu sync.Mutex
	// pending holds the latest serialized state per file, written by
	// the writer goroutine; later saves replace earlier ones.
	pendMu  sync.Mutex
	pending map[string][]byte
	kick    chan struct{}
	idle    sync.WaitGroup

	memMu  sync.Mutex
	memory []string

	// vault holds API keys when the credential store works; keys and
	// settingsFull (settings.json with the keys, the fallback) are
	// written with the pending batch.
	vault        *secretVault
	keys         map[string]string
	settingsFull []byte
}

// Open opens (creating) the data directory.
func Open(dir string) (*Store, error) {
	for _, sub := range []string{"conversations", "attachments", "workspaces", "models"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return nil, err
		}
	}
	s := &Store{dir: dir, pending: map[string][]byte{}, kick: make(chan struct{}, 1)}
	_ = readJSON(filepath.Join(dir, "memory.json"), &s.memory)
	go s.writer()
	return s, nil
}

// Dir is the data directory.
func (s *Store) Dir() string { return s.dir }

// ModelsDir holds downloaded models.
func (s *Store) ModelsDir() string { return filepath.Join(s.dir, "models") }

// Workspace returns (creating) a conversation's agent workspace.
func (s *Store) Workspace(convID string) string {
	d := filepath.Join(s.dir, "workspaces", convID)
	_ = os.MkdirAll(d, 0o755)
	return d
}

// AttachmentPath is where an attachment's bytes live.
func (s *Store) AttachmentPath(id string) string { return filepath.Join(s.dir, "attachments", id) }

// AttachmentTextPath is where a document's extracted text lives.
func (s *Store) AttachmentTextPath(id string) string {
	return filepath.Join(s.dir, "attachments", id+".txt")
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// writeFile writes atomically through a temp file.
func writeFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) queue(path string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	s.pendMu.Lock()
	if _, ok := s.pending[path]; !ok {
		s.idle.Add(1)
	}
	s.pending[path] = data
	s.pendMu.Unlock()
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

func (s *Store) writer() {
	for range s.kick {
		// Coalesce bursts (a streaming reply saves often).
		time.Sleep(150 * time.Millisecond)
		s.flushPending()
	}
}

func (s *Store) flushPending() {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.pendMu.Lock()
	batch := s.pending
	s.pending = map[string][]byte{}
	keys, full := s.keys, s.settingsFull
	s.keys, s.settingsFull = nil, nil
	s.pendMu.Unlock()
	if keys != nil && !s.vault.write(keys) {
		batch[s.settingsPath()] = full
	}
	for path, data := range batch {
		_ = writeFile(path, data)
		s.idle.Done()
	}
}

// Flush writes everything queued and waits for it - call it when the
// app goes to the background.
func (s *Store) Flush() {
	s.flushPending()
	s.idle.Wait()
}

// LoadSettings reads the settings, or returns defaults.
func (s *Store) LoadSettings() *Settings {
	st := DefaultSettings()
	if err := readJSON(s.settingsPath(), st); err != nil {
		st = DefaultSettings()
	}
	if st.AutoApprove == nil {
		st.AutoApprove = map[string]bool{}
	}
	if s.vault != nil {
		s.vault.fillKeys(st)
	}
	return st
}

func (s *Store) settingsPath() string { return filepath.Join(s.dir, "settings.json") }

// SaveSettings persists st, with its API keys in the credential store
// when UseSecrets enabled it.
func (s *Store) SaveSettings(st *Settings) {
	if s.vault == nil {
		s.queue(s.settingsPath(), st)
		return
	}
	full, err := json.Marshal(st)
	if err != nil {
		return
	}
	s.pendMu.Lock()
	s.keys, s.settingsFull = keysOf(st), full
	s.pendMu.Unlock()
	s.queue(s.settingsPath(), withoutKeys(st))
}

// LoadAgents reads custom agents merged after the built-in ones.
func (s *Store) LoadAgents() []*Agent {
	var custom []*Agent
	_ = readJSON(filepath.Join(s.dir, "agents.json"), &custom)
	out := BuiltinAgents()
	return append(out, custom...)
}

// SaveAgents persists the custom agents of all.
func (s *Store) SaveAgents(all []*Agent) {
	var custom []*Agent
	for _, a := range all {
		if !a.Builtin {
			custom = append(custom, a)
		}
	}
	s.queue(filepath.Join(s.dir, "agents.json"), custom)
}

// LoadConversations reads every conversation, most recent first.
func (s *Store) LoadConversations() []*Conversation {
	dir := filepath.Join(s.dir, "conversations")
	entries, _ := os.ReadDir(dir)
	var out []*Conversation
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		c := &Conversation{}
		if readJSON(filepath.Join(dir, e.Name()), c) == nil && c.ID != "" {
			// A reply cut off by the app being killed.
			for _, m := range c.Messages {
				for i := range m.Blocks {
					if t := m.Blocks[i].Tool; t != nil && (t.Status == ToolRunning || t.Status == ToolPending) {
						t.Status = ToolCanceled
					}
				}
			}
			out = append(out, c)
		}
	}
	SortConversations(out)
	return out
}

// SortConversations orders pinned first, then by last update.
func SortConversations(cs []*Conversation) {
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].Pinned != cs[j].Pinned {
			return cs[i].Pinned
		}
		return cs[i].Updated.After(cs[j].Updated)
	})
}

// SaveConversation persists c.
func (s *Store) SaveConversation(c *Conversation) {
	s.queue(filepath.Join(s.dir, "conversations", c.ID+".json"), c)
}

// DeleteConversation removes c and its attachments and workspace.
func (s *Store) DeleteConversation(c *Conversation) {
	go func() {
		s.Flush()
		_ = os.Remove(filepath.Join(s.dir, "conversations", c.ID+".json"))
		_ = os.RemoveAll(filepath.Join(s.dir, "workspaces", c.ID))
		for _, m := range c.Messages {
			for _, a := range m.Attachments {
				_ = os.Remove(s.AttachmentPath(a.ID))
				_ = os.Remove(s.AttachmentTextPath(a.ID))
			}
		}
	}()
}

// DeleteAll erases every conversation, attachment and workspace.
func (s *Store) DeleteAll() {
	s.Flush()
	for _, sub := range []string{"conversations", "attachments", "workspaces"} {
		_ = os.RemoveAll(filepath.Join(s.dir, sub))
		_ = os.MkdirAll(filepath.Join(s.dir, sub), 0o755)
	}
}

// Remember implements tools.Memory.
func (s *Store) Remember(fact string) error {
	fact = strings.TrimSpace(fact)
	if fact == "" {
		return errors.New("empty fact")
	}
	s.memMu.Lock()
	defer s.memMu.Unlock()
	if slices.Contains(s.memory, fact) {
		return nil
	}
	if len(s.memory) >= 200 {
		return errors.New("memory is full - ask the user to remove old memories in Settings")
	}
	s.memory = append(s.memory, fact)
	s.queue(filepath.Join(s.dir, "memory.json"), slices.Clone(s.memory))
	return nil
}

// Forget implements tools.Memory.
func (s *Store) Forget(fact string) bool {
	s.memMu.Lock()
	defer s.memMu.Unlock()
	for i, f := range s.memory {
		if strings.EqualFold(strings.TrimSpace(f), strings.TrimSpace(fact)) {
			s.memory = slices.Delete(s.memory, i, i+1)
			s.queue(filepath.Join(s.dir, "memory.json"), slices.Clone(s.memory))
			return true
		}
	}
	return false
}

// All implements tools.Memory.
func (s *Store) All() []string {
	s.memMu.Lock()
	defer s.memMu.Unlock()
	return slices.Clone(s.memory)
}

// ClearMemory forgets everything.
func (s *Store) ClearMemory() {
	s.memMu.Lock()
	s.memory = nil
	s.memMu.Unlock()
	s.queue(filepath.Join(s.dir, "memory.json"), []string{})
}
