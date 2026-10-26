package ui

import (
	"context"
	"strings"
	"time"

	"github.com/mehran9675/pocket-agent/internal/agent"
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// generateTitle asks the model for a short title.
func (a *App) generateTitle(c *store.Conversation, prov llm.Provider, ref store.ModelRef) {
	if len(c.Messages) < 2 {
		return
	}
	q := c.Messages[0].Text
	reply := c.Messages[1].ReplyText()
	if r := []rune(reply); len(r) > 600 {
		reply = string(r[:600])
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		title, err := agent.Complete(ctx, prov, ref.Model, "You write concise conversation titles.",
			"Write a title of 2-6 words for this conversation. Reply with the title only, no quotes or punctuation at the end.\n\nUser: "+q+"\nAssistant: "+reply, 60)
		if err != nil {
			return
		}
		title = strings.Trim(strings.TrimSpace(strings.Split(strings.TrimSpace(title), "\n")[0]), `"'*#.`)
		if title == "" || len([]rune(title)) > 60 {
			return
		}
		a.Post(func() {
			c.Title = title
			c.Version++
			a.st.SaveConversation(c)
		})
	}()
}
