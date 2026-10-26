package agent

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// inlineChars is how much of a document goes straight into the prompt;
// the rest is reachable with read_attachment.
const inlineChars = 40000

// BuildHistory converts a conversation's messages to model messages.
// Images are only sent when vision is set.
func BuildHistory(st *store.Store, msgs []*store.Message, vision bool) []llm.Message {
	var out []llm.Message
	for _, m := range msgs {
		switch m.Role {
		case llm.RoleUser:
			out = append(out, userMessage(st, m, vision))
		case llm.RoleAssistant:
			out = append(out, assistantMessages(m)...)
		}
	}
	return out
}

func userMessage(st *store.Store, m *store.Message, vision bool) llm.Message {
	msg := llm.Message{Role: llm.RoleUser}
	var docs strings.Builder
	for _, a := range m.Attachments {
		switch a.Kind {
		case "image":
			if !vision {
				fmt.Fprintf(&docs, "[Image %q attached; the current model can't view images.]\n", a.Name)
				continue
			}
			data, err := os.ReadFile(st.AttachmentPath(a.ID))
			if err != nil {
				continue
			}
			msg.Parts = append(msg.Parts, llm.Part{Kind: llm.PartImage, MIME: a.MIME, Data: data})
		default:
			text, err := os.ReadFile(st.AttachmentTextPath(a.ID))
			if err != nil {
				continue
			}
			r := []rune(string(text))
			fmt.Fprintf(&docs, "<attachment name=%q", a.Name)
			if a.Pages > 0 {
				fmt.Fprintf(&docs, " pages=\"%d\"", a.Pages)
			}
			docs.WriteString(">\n")
			if len(r) > inlineChars {
				docs.WriteString(string(r[:inlineChars]))
				fmt.Fprintf(&docs, "\n[... %d more characters - use read_attachment with name=%q and offset=%d to read on]", len(r)-inlineChars, a.Name, inlineChars)
			} else {
				docs.WriteString(string(r))
			}
			docs.WriteString("\n</attachment>\n\n")
		}
	}
	text := m.Text
	if docs.Len() > 0 {
		text = docs.String() + text
	}
	if strings.TrimSpace(text) == "" && len(msg.Parts) > 0 {
		text = "(see the attached image)"
	}
	msg.Parts = append(msg.Parts, llm.TextPart(text))
	return msg
}

// assistantMessages expands a turn's blocks into the assistant/tool
// message sequence the model produced: text and tool calls, then the
// results, then more text...
func assistantMessages(m *store.Message) []llm.Message {
	var out []llm.Message
	var cur llm.Message
	var results []llm.Message
	flush := func() {
		if cur.Text() == "" && len(cur.ToolCalls) == 0 {
			return
		}
		out = append(out, cur)
		out = append(out, results...)
		cur, results = llm.Message{}, nil
	}
	cur.Role = llm.RoleAssistant
	for _, b := range m.Blocks {
		switch b.Kind {
		case store.BlockText:
			if len(cur.ToolCalls) > 0 {
				flush()
				cur.Role = llm.RoleAssistant
			}
			if strings.TrimSpace(b.Text) != "" {
				cur.Parts = append(cur.Parts, llm.TextPart(b.Text))
			}
		case store.BlockTool:
			t := b.Tool
			if t == nil {
				continue
			}
			args := t.Args
			if args == "" {
				args = "{}"
			}
			cur.ToolCalls = append(cur.ToolCalls, llm.ToolCall{ID: t.CallID, Name: t.Name, Args: []byte(args)})
			result := t.Result
			switch t.Status {
			case store.ToolDenied:
				result = "The user declined this tool call."
			case store.ToolCanceled, store.ToolPending, store.ToolRunning:
				result = "The tool call was canceled."
			}
			results = append(results, llm.Message{
				Role: llm.RoleTool, ToolCallID: t.CallID, ToolName: t.Name,
				Parts: []llm.Part{llm.TextPart(result)}, IsError: t.IsError || t.Status != store.ToolDone,
			})
		}
	}
	cur.Role = llm.RoleAssistant
	flush()
	if len(out) == 0 && m.Error == "" {
		out = append(out, llm.Message{Role: llm.RoleAssistant, Parts: []llm.Part{llm.TextPart("(no reply)")}})
	}
	// A turn that ends on tool results needs a closing assistant text
	// for providers that require alternation.
	if n := len(out); n > 0 && out[n-1].Role == llm.RoleTool {
		out = append(out, llm.Message{Role: llm.RoleAssistant, Parts: []llm.Part{llm.TextPart("(stopped)")}})
	}
	return out
}

// SystemPromptInput feeds SystemPrompt.
type SystemPromptInput struct {
	Instructions string // the user's custom instructions
	Override     string // per-conversation system prompt
	Memory       []string
	Agent        *store.Agent
	Mode         store.Mode
	WebSearch    bool
	HasTools     bool
	Local        bool
	Workspace    []string // existing workspace files
}

// SystemPrompt composes the system prompt for a turn.
func SystemPrompt(in SystemPromptInput) string {
	var b strings.Builder
	now := time.Now()
	if in.Override != "" {
		b.WriteString(in.Override)
	} else if in.Agent != nil {
		b.WriteString(in.Agent.Instructions)
	} else {
		b.WriteString("You are Pocket Agent, a capable, candid AI assistant running in a mobile app. ")
		b.WriteString("Answer directly and precisely. Use Markdown - headings, lists, tables and fenced code blocks - when it aids readability; keep short answers short. ")
		b.WriteString("The screen may be narrow: prefer compact tables and short paragraphs.")
	}
	fmt.Fprintf(&b, "\n\nCurrent date: %s.", now.Format("Monday, 2 January 2006, 15:04 MST"))
	if in.WebSearch || in.Mode == store.ModeAgent {
		b.WriteString(" Your training data has a cutoff; use web_search for anything that may have changed, and cite sources as markdown links.")
	}
	if in.Mode == store.ModeAgent {
		b.WriteString("\n\nYou are running in agent mode: work autonomously through the task with your tools until it is done, " +
			"without asking for confirmation at each step. Files you write go to a private workspace the user can open and share.")
		if len(in.Workspace) > 0 {
			b.WriteString(" Current workspace files: " + strings.Join(in.Workspace, ", ") + ".")
		}
	}
	if in.Instructions != "" {
		b.WriteString("\n\n# The user's custom instructions\n" + in.Instructions)
	}
	if len(in.Memory) > 0 {
		b.WriteString("\n\n# What you remember about the user\n")
		for _, f := range in.Memory {
			b.WriteString("- " + f + "\n")
		}
		b.WriteString("Use these naturally when relevant; don't recite them.")
	}
	return b.String()
}
