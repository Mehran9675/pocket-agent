package ui

import (
	"context"

	"github.com/mehran9675/pocket-agent/internal/agent"
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// summaryFrom returns where c's unsummarized messages start and the
// summary of those before; 0 and "" when there is none, or when an edit
// removed the message it ran through.
func summaryFrom(c *store.Conversation) (int, string) {
	if c.Summary == "" {
		return 0, ""
	}
	for i, m := range c.Messages {
		if m.ID == c.SummaryThrough {
			return i + 1, c.Summary
		}
	}
	return 0, ""
}

// compressionLimit is the context size, in tokens, past which earlier
// messages are summarized; 0 when compression is off. On-device models
// compress before their context window fills.
func (a *App) compressionLimit(ref store.ModelRef) int {
	if !a.settings.Compression.Enabled {
		return 0
	}
	n := a.settings.Compression.Tokens()
	if ref.Provider == store.LocalProviderID && a.engine != nil {
		if w := a.engine.ContextSize(); w > 0 {
			n = min(n, w*6/10)
		}
	}
	return n
}

// keptFrom is the index of the first message kept word for word when
// msgs[from:] is compressed: the previous exchange stays, so the reply
// follows on naturally.
func keptFrom(msgs []*store.Message, from int) int {
	users := 0
	for i := len(msgs) - 1; i >= from; i-- {
		if msgs[i].Role == llm.RoleUser {
			if users++; users == 2 {
				return i
			}
		}
	}
	return from
}

// compressHistory summarizes the start of a turn's history when it has
// grown past the limit, and returns the history and summary to send.
// msgs are the conversation's messages before the reply. It runs off
// the UI goroutine; the conversation is updated through Post.
func (a *App) compressHistory(ctx context.Context, c *store.Conversation, reply *store.Message, prov llm.Provider, model string, vision bool,
	msgs []*store.Message, from int, summary string, prompts agent.Prompts, limit int, obs *observer) ([]llm.Message, string) {
	history := agent.BuildHistory(a.st, msgs[from:], vision)
	if limit <= 0 || agent.EstimateTokens(summary, history) <= limit {
		return history, summary
	}
	cut := keptFrom(msgs, from)
	if cut <= from {
		return history, summary
	}
	sum, u, err := agent.Summarize(ctx, prov, model, prompts.Get(agent.PromptSummary), summary,
		agent.BuildHistory(a.st, msgs[from:cut], false), llm.ThinkingAuto)
	if err != nil {
		return history, summary
	}
	obs.Usage(u)
	through := msgs[cut-1].ID
	a.Post(func() {
		c.Summary, c.SummaryThrough = sum, through
		reply.Compacted = true
		reply.Version++
		a.st.SaveConversation(c)
	})
	return agent.BuildHistory(a.st, msgs[cut:], vision), sum
}
