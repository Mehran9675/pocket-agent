package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/agent"
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

// send starts a turn with the composer's content.
func (a *App) send(text string, atts []store.Attachment) {
	text = strings.TrimSpace(text)
	if text == "" && len(atts) == 0 {
		return
	}
	ref := a.currentModel()
	if ref.IsZero() {
		a.openModelSheet()
		a.toast(giode.ToastInfo, "Choose a model to start")
		return
	}
	c := a.current
	if c == nil {
		c = &store.Conversation{
			ID:        store.NewID(),
			Created:   time.Now(),
			Model:     ref,
			Mode:      a.draft.mode,
			AgentID:   a.draft.agentID,
			WebSearch: a.draft.web,
			Depth:     a.draft.depth,
			Thinking:  a.settings.Thinking,
		}
		c.Title = titleFromText(text)
		a.convs = append([]*store.Conversation{c}, a.convs...)
		a.current = c
	}
	if a.isRunning(c) {
		return
	}
	c.Messages = append(c.Messages, &store.Message{
		ID: store.NewID(), Role: llm.RoleUser, Created: time.Now(), Text: text, Attachments: atts,
	})
	c.Touch()
	store.SortConversations(a.convs)
	a.chat.jumpToBottom = true
	a.startTurn(c)
}

// startTurn appends an assistant message and runs the model for c.
func (a *App) startTurn(c *store.Conversation) {
	ref := c.Model
	if ref.IsZero() {
		ref = a.currentModel()
		c.Model = ref
	}
	prov, info, err := a.resolve(ref)
	name, _ := a.modelLabel(ref)
	msg := &store.Message{ID: store.NewID(), Role: llm.RoleAssistant, Created: time.Now(), Model: name, Streaming: true}
	c.Messages = append(c.Messages, msg)
	c.Touch()
	if err != nil {
		msg.Streaming = false
		msg.Error = agent.FriendlyError(err)
		a.st.SaveConversation(c)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	rs := &runState{cancel: cancel, approvals: map[string]chan bool{}}
	a.runs[c.ID] = rs
	a.runStarted(ref)
	a.st.SaveConversation(c)

	history := agent.BuildHistory(a.st, c.Messages[:len(c.Messages)-1], info.Vision)
	obs := &observer{a: a, c: c, m: msg, rs: rs}
	thinking := c.Thinking
	if !a.modelThinks(ref) {
		thinking = llm.ThinkingAuto
	}
	started := time.Now()
	firstTurn := len(c.Messages) <= 2

	switch c.Mode {
	case store.ModeResearch:
		question := ""
		if n := len(c.Messages); n >= 2 {
			question = c.Messages[n-2].Text
		}
		msg.Research = &store.Research{Phase: store.PhasePlanning, Started: started}
		var prior strings.Builder
		for _, m := range c.Messages[:max(0, len(c.Messages)-2)] {
			if m.Role == llm.RoleUser {
				prior.WriteString("User: " + m.Text + "\n")
			} else if t := m.ReplyText(); t != "" {
				r := []rune(t)
				if len(r) > 1500 {
					t = string(r[:1500]) + "…"
				}
				prior.WriteString("Assistant: " + t + "\n")
			}
		}
		r := &agent.Research{
			Provider: prov, Model: ref.Model, Question: question, Context: prior.String(),
			Depth: c.Depth, Search: a.settings.Search, Observer: obs,
			Small: ref.Provider == store.LocalProviderID, Thinking: thinking,
		}
		go func() {
			err := r.Run(ctx)
			a.Post(func() { a.finishTurn(c, msg, rs, err, started, firstTurn, prov, ref) })
		}()
	default:
		ag := a.agentByID(c.AgentID)
		if c.Mode != store.ModeAgent {
			ag = nil
		}
		set := a.toolSet(c, ag)
		env := &tools.Env{
			Search: a.settings.Search,
			Attachments: func() []tools.Attachment {
				return a.attachmentTexts(c)
			},
			OnPlan: func(items []tools.PlanItem) {
				obs.plan(items)
			},
		}
		if a.settings.MemoryEnabled {
			env.Memory = a.st
		}
		if c.Mode == store.ModeAgent {
			env.Workspace = a.st.Workspace(c.ID)
		}
		var memory []string
		if a.settings.MemoryEnabled {
			memory = a.st.All()
		}
		maxSteps := 10
		if ag != nil {
			maxSteps = ag.MaxSteps
		}
		system := agent.SystemPrompt(agent.SystemPromptInput{
			Instructions: a.settings.Instructions,
			Override:     c.System,
			Memory:       memory,
			Agent:        ag,
			Mode:         c.Mode,
			WebSearch:    c.WebSearch,
			HasTools:     set.Len() > 0,
			Local:        ref.Provider == store.LocalProviderID,
			Workspace:    a.workspaceFiles(c),
			MaxSteps:     maxSteps,
			TokenBudget:  ag.Budget(),
		})
		if !info.Tools {
			set = tools.NewSet()
		}
		run := &agent.Run{
			Provider: prov, Model: ref.Model, System: system, Messages: history,
			Tools: set, Env: env, MaxSteps: maxSteps, TokenBudget: ag.Budget(),
			Reasoning: a.settings.ShowReasoning && info.Reasoning,
			Thinking:  thinking,
			Observer:  obs,
			NeedsApproval: func(t *tools.Tool) bool {
				if !t.Approval || a.settings.AutoApprove[t.Name] {
					return false
				}
				return ag == nil || !ag.AutoApprove
			},
			Approve: func(ctx context.Context, call llm.ToolCall, t *tools.Tool) bool {
				return obs.approve(ctx, call)
			},
		}
		if ref.Provider == store.LocalProviderID {
			run.MaxTokens = a.settings.Local.MaxTokens
		}
		go func() {
			_, err := run.Execute(ctx)
			a.Post(func() { a.finishTurn(c, msg, rs, err, started, firstTurn, prov, ref) })
		}()
	}
}

func (a *App) finishTurn(c *store.Conversation, msg *store.Message, rs *runState, err error, started time.Time, firstTurn bool, prov llm.Provider, ref store.ModelRef) {
	if a.runs[c.ID] == rs {
		delete(a.runs, c.ID)
	}
	defer a.runEnded(c, msg, ref)
	msg.Streaming = false
	msg.Duration = time.Since(started)
	for i := range msg.Blocks {
		if t := msg.Blocks[i].Tool; t != nil && (t.Status == store.ToolRunning || t.Status == store.ToolPending) {
			t.Status = store.ToolCanceled
		}
	}
	switch {
	case err == nil:
	case errors.Is(err, context.Canceled):
		msg.Stopped = true
	case errors.Is(err, agent.ErrMaxSteps):
		msg.Error = "The agent paused after reaching its step limit."
		msg.Paused = true
	case errors.Is(err, agent.ErrTokenBudget):
		msg.Error = fmt.Sprintf("The agent paused after using %s tokens, its budget for one task.", compactNum(msg.Usage.InputTokens+msg.Usage.OutputTokens))
		msg.Paused = true
	default:
		msg.Error = agent.FriendlyError(err)
	}
	msg.Version++
	c.Touch()
	a.st.SaveConversation(c)
	if err == nil && firstTurn && ref.Provider != store.LocalProviderID {
		a.generateTitle(c, prov, ref)
	}
}
