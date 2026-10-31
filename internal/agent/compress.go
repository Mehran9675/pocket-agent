package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/mehran9675/pocket-agent/internal/llm"
)

// EstimateTokens guesses the tokens system and msgs take: four
// characters a token, as most tokenizers average on English.
func EstimateTokens(system string, msgs []llm.Message) int {
	chars := len(system)
	for _, m := range msgs {
		chars += len(m.Text())
		for _, c := range m.ToolCalls {
			chars += len(c.Name) + len(c.Args)
		}
		for _, p := range m.Parts {
			if p.Kind == llm.PartImage {
				chars += 4 * 1000
			}
		}
	}
	return chars / 4
}

// transcriptResult caps each tool result in a transcript.
const transcriptResult = 2500

// transcript writes msgs as plain text for a summary.
func transcript(msgs []llm.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		switch m.Role {
		case llm.RoleUser:
			b.WriteString("User: " + m.Text() + "\n\n")
		case llm.RoleAssistant:
			if t := strings.TrimSpace(m.Text()); t != "" {
				b.WriteString("Assistant: " + t + "\n\n")
			}
			for _, c := range m.ToolCalls {
				fmt.Fprintf(&b, "Assistant called %s %s\n\n", c.Name, c.Args)
			}
		case llm.RoleTool:
			r := []rune(m.Text())
			if len(r) > transcriptResult {
				r = append(r[:transcriptResult], []rune(" […]")...)
			}
			fmt.Fprintf(&b, "Result of %s: %s\n\n", m.ToolName, string(r))
		}
	}
	return b.String()
}

// Summarize condenses msgs - continuing prior, an earlier summary -
// with system as the instructions. It returns the summary and the
// tokens the call used.
func Summarize(ctx context.Context, p llm.Provider, model, system, prior string, msgs []llm.Message, thinking llm.Thinking) (string, llm.Usage, error) {
	var b strings.Builder
	if prior != "" {
		b.WriteString("Summary of what came before:\n" + prior + "\n\n")
	}
	b.WriteString("Conversation to condense:\n\n" + transcript(msgs))
	b.WriteString("\nWrite the condensed summary now.")
	resp, err := chat(ctx, p, llm.Request{
		Model:     model,
		System:    system,
		Messages:  []llm.Message{{Role: llm.RoleUser, Parts: []llm.Part{llm.TextPart(b.String())}}},
		MaxTokens: 1200,
		Thinking:  thinking,
	}, nil)
	if err != nil {
		return "", llm.Usage{}, err
	}
	s := strings.TrimSpace(resp.Text)
	if s == "" {
		return "", resp.Usage, fmt.Errorf("the summary came back empty")
	}
	return s, resp.Usage, nil
}
