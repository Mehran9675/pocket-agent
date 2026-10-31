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
		a.toast(giode.ToastInfo, tr("Choose a model to start"))
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

// wrapUpPrompt asks a paused agent for its results.
const wrapUpPrompt = "Stop here and write up your results now from what you have so far. Don't call any more tools; say briefly what is left undone."

// wrapUp asks the paused agent in c to finish with what it has.
func (a *App) wrapUp(c *store.Conversation) {
	if a.isRunning(c) {
		return
	}
	c.Messages = append(c.Messages, &store.Message{
		ID: store.NewID(), Role: llm.RoleUser, Created: time.Now(), Text: wrapUpPrompt, WrapUp: true,
	})
	c.Touch()
	a.chat.jumpToBottom = true
	a.startTurn(c)
}

func hasTool(set *tools.Set, name string) bool {
	_, ok := set.Get(name)
	return ok
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
		msg.Error = tr(agent.FriendlyError(err))
		a.st.SaveConversation(c)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	rs := &runState{cancel: cancel, approvals: map[string]chan bool{}}
	a.runs[c.ID] = rs
	a.runStarted(ref)
	a.st.SaveConversation(c)

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
			Prompts: agent.Prompts(a.settings.Prompts),
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
			Skills: skillServer{a.skills, a.enabledSkills()},
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
		prompts := agent.Prompts(a.settings.Prompts)
		from, summary := summaryFrom(c)
		sysIn := agent.SystemPromptInput{
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
			Skills:       a.skillIndex(set, info.Tools),
			Prompts:      prompts,
		}
		if !info.Tools {
			set = tools.NewSet()
		}
		run := &agent.Run{
			Provider: prov, Model: ref.Model,
			Tools: set, Env: env, MaxSteps: maxSteps, TokenBudget: ag.Budget(),
			Reasoning: a.settings.ShowReasoning && info.Reasoning,
			Thinking:  thinking,
			Observer:  obs,
			Prompts:   prompts,
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
		if n := len(c.Messages); n >= 2 && c.Messages[n-2].WrapUp {
			run.Final = true
		}
		limit := a.compressionLimit(ref)
		run.CompressAt = limit
		// Within a task every step resends the context, so agents
		// condense earlier, keeping a budget good for several steps.
		if b := ag.Budget(); limit > 0 && c.Mode == store.ModeAgent && b > 0 {
			run.CompressAt = min(limit, max(b/8, 16000))
		}
		msgs := append([]*store.Message(nil), c.Messages[:len(c.Messages)-1]...)
		go func() {
			run.Messages, sysIn.Summary = a.compressHistory(ctx, c, msg, prov, ref.Model, info.Vision, msgs, from, summary, prompts, limit, obs)
			run.System = agent.SystemPrompt(sysIn)
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
		msg.Error = tr("The agent paused after reaching its step limit. Continue to let it keep working, or wrap up to get its results now.")
		msg.Paused = true
	case errors.Is(err, agent.ErrTokenBudget):
		msg.Error = fmt.Sprintf(tr("The agent paused after using %s tokens, its budget for one task. Continue to give it another budget, or wrap up to get its results now."), compactNum(msg.Usage.InputTokens+msg.Usage.OutputTokens))
		msg.Paused = true
	default:
		msg.Error = tr(agent.FriendlyError(err))
	}
	msg.Version++
	c.Touch()
	a.st.SaveConversation(c)
	if err == nil && firstTurn && ref.Provider != store.LocalProviderID {
		a.generateTitle(c, prov, ref)
	}
}
