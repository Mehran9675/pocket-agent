//go:build devmock

package llm

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// The devmock build adds an offline "Demo" provider that streams
// scripted replies, tool calls and research steps, for developing and
// screenshotting the UI without API keys.
func init() {
	mockFactory = func() Provider { return mock{} }
	Presets = append([]Preset{{
		ID: "demo", Name: "Demo (offline)", Kind: kindMock, KeyOptional: true,
		KeyHint: "Scripted replies for UI development",
		Models:  []ModelInfo{{ID: "pocket-demo", Name: "Pocket Demo", Vision: true, Tools: true, Reasoning: true, ContextWindow: 200000}},
	}}, Presets...)
}

type mock struct{}

func (mock) Models(context.Context) ([]ModelInfo, error) {
	return []ModelInfo{{ID: "pocket-demo", Name: "Pocket Demo", Vision: true, Tools: true, Reasoning: true, ContextWindow: 200000}}, nil
}

func stream(ctx context.Context, col *collector, kind EventKind, s string) error {
	words := strings.SplitAfter(s, " ")
	for _, w := range words {
		if err := ctx.Err(); err != nil {
			return err
		}
		if kind == EventReasoning {
			col.reasoningDelta(w)
		} else {
			col.textDelta(w)
		}
		time.Sleep(12 * time.Millisecond)
	}
	return nil
}

func (mock) Chat(ctx context.Context, req Request, onEvent func(Event)) (*Response, error) {
	col := newCollector(onEvent)
	sys := req.System
	last := ""
	toolResults, steps := 0, 0
	for i := len(req.Messages) - 1; i >= 0; i-- {
		m := req.Messages[i]
		if m.Role == RoleTool {
			toolResults++
			continue
		}
		if m.Role == RoleAssistant {
			steps++
			continue
		}
		if m.Role == RoleUser {
			last = m.Text()
			break
		}
	}
	has := func(name string) bool {
		for _, t := range req.Tools {
			if t.Name == name {
				return true
			}
		}
		return false
	}
	call := func(name string, args any) {
		b, _ := json.Marshal(args)
		col.toolCall(ToolCall{ID: newCallID(), Name: name, Args: b})
	}
	switch {
	case strings.Contains(sys, "You plan web research"):
		time.Sleep(700 * time.Millisecond)
		col.textDelta(`{"goal": "Assess how close solid-state batteries are to mass-market EVs, covering technology, costs and timelines.",
"queries": [{"question": "Which manufacturers have credible production timelines?", "query": "solid-state battery production timeline 2027 Toyota Samsung"},
{"question": "What energy density and cycle life do current samples reach?", "query": "solid-state battery energy density cycle life samples 2026"},
{"question": "How do costs compare with lithium-ion?", "query": "solid-state battery cost per kWh vs lithium-ion"},
{"question": "What technical hurdles remain?", "query": "solid electrolyte dendrite interface stability challenges"}]}`)
	case strings.Contains(sys, "You extract research notes"):
		time.Sleep(900 * time.Millisecond)
		col.textDelta("- Pilot lines target 400-500 Wh/kg cells for 2027-2028 vehicles\n- Pilot-scale cost is roughly 2-3x conventional lithium-ion\n- Interface stability and manufacturing yield remain the main hurdles")
	case strings.Contains(sys, "You review research progress"):
		col.textDelta(`{"done": true, "queries": []}`)
	case strings.Contains(sys, "expert research analyst"):
		err := stream(ctx, col, EventText, mockReport)
		if err != nil {
			return nil, err
		}
	case strings.Contains(sys, "concise conversation titles"):
		col.textDelta("Solid-State Battery Outlook")
	case strings.Contains(strings.ToLower(last), "delete") && has("delete_file") && toolResults == 0:
		call("delete_file", map[string]string{"path": "draft.md"})
	case has("update_plan"):
		if err := mockAgent(ctx, col, steps, call); err != nil {
			return nil, err
		}
	case has("web_search") && toolResults == 0:
		_ = stream(ctx, col, EventReasoning, "The user wants current information, so I should search the web before answering.")
		call("web_search", map[string]any{"query": last})
	default:
		if req.Reasoning {
			if err := stream(ctx, col, EventReasoning, "Let me structure this: a short overview, a comparison table, then a code sample and practical next steps."); err != nil {
				return nil, err
			}
		}
		if err := stream(ctx, col, EventText, mockAnswer); err != nil {
			return nil, err
		}
	}
	col.resp.Usage = Usage{InputTokens: 1840, OutputTokens: 612}
	return col.finish(), nil
}

func mockAgent(ctx context.Context, col *collector, step int, call func(string, any)) error {
	plan := func(statuses ...string) []map[string]string {
		steps := []string{"Research the leading options", "Compare specs and prices", "Write the buying guide to guide.md"}
		var out []map[string]string
		for i, s := range steps {
			out = append(out, map[string]string{"step": s, "status": statuses[i]})
		}
		return out
	}
	switch step {
	case 0:
		_ = stream(ctx, col, EventText, "I'll research the options, compare them and write a guide. ")
		call("update_plan", map[string]any{"steps": plan("in_progress", "pending", "pending")})
	case 1:
		call("web_search", map[string]any{"query": "best budget mechanical keyboard 2026"})
	case 2:
		call("update_plan", map[string]any{"steps": plan("completed", "in_progress", "pending")})
		call("run_javascript", map[string]any{"code": "const k=[{n:'Keychron V1',p:79},{n:'Akko 5075B',p:69},{n:'Royal Kludge R65',p:59}]; k.map(x=>x.n+': $'+x.p).join('\\n')"})
	case 3:
		call("write_file", map[string]any{"path": "guide.md", "content": "# Budget mechanical keyboards\n\n| Model | Price | Why |\n|---|---|---|\n| Keychron V1 | $79 | Best overall |\n| Akko 5075B | $69 | Best typing feel |\n| Royal Kludge R65 | $59 | Best value |\n"})
	case 4:
		call("update_plan", map[string]any{"steps": plan("completed", "completed", "completed")})
	default:
		return stream(ctx, col, EventText, "Done. I compared three keyboards and saved a buying guide to **guide.md**:\n\n| Model | Price | Best for |\n|---|---|---|\n| Keychron V1 | $79 | Overall |\n| Akko 5075B | $69 | Typing feel |\n| Royal Kludge R65 | $59 | Value |\n\nOpen the file below to read or share it.")
	}
	return nil
}

const mockAnswer = "## Running models on a phone\n\n" +
	"On-device inference trades **raw capability** for *privacy, offline use and zero cost per token*. " +
	"Small models (1-4B parameters) quantized to 4 bits fit comfortably in 3-8 GB of RAM.\n\n" +
	"| Model | Size | Good for |\n|---|---|---|\n| Qwen3 1.7B | 1.1 GB | Reasoning, tools |\n| Llama 3.2 3B | 2.0 GB | Writing, summaries |\n| Gemma 3 4B | 2.5 GB | Multilingual |\n\n" +
	"A minimal `llama.cpp` sampling loop looks like this:\n\n" +
	"```go\nfor n := 0; n < maxTokens; n++ {\n\ttok := llama.Sample(ctx)\n\tif vocab.IsEOG(tok) {\n\t\tbreak\n\t}\n\temit(vocab.Piece(tok))\n}\n```\n\n" +
	"### Practical tips\n\n1. Prefer **Q4_K_M** quantization for the best size/quality balance.\n2. Keep the context at 4K tokens on phones with 6 GB of RAM.\n3. Use a cloud model for long documents and deep research.\n\n" +
	"> Tip: you can switch models mid-conversation from the picker at the top.\n\n" +
	"Read more in the [llama.cpp docs](https://github.com/ggml-org/llama.cpp)."

const mockReport = "Solid-state batteries are moving from lab to pilot production, but mass-market EVs using them are unlikely before 2028-2030. " +
	"Leading manufacturers report 400-500 Wh/kg samples, while costs remain two to three times higher than lithium-ion at pilot scale [[1]](https://www.example-energy.org/solid-state-2026).\n\n" +
	"## Production timelines\n\n" +
	"Toyota plans limited sulfide-electrolyte production for 2027-2028 vehicles [[2]](https://news.example-auto.com/toyota-solid-state), and Samsung SDI's pilot line has entered validation [[6]](https://www.example-korea-tech.com/sdi-pilot).\n\n" +
	"| Company | Chemistry | Target |\n|---|---|---|\n| Toyota | Sulfide | 2027-28 |\n| QuantumScape | Oxide separator | B-samples now |\n| Samsung SDI | Sulfide | 2027 |\n\n" +
	"## Remaining hurdles\n\n" +
	"- **Interface stability** between lithium metal and electrolyte limits cycle life [[4]](https://journals.example-science.org/sse-review)\n- **Manufacturing yield** at gigafactory scale is unproven\n- **Cost** depends on electrolyte materials and dry-room processing [[5]](https://www.example-bnef.com/battery-costs)\n\n" +
	"## Key takeaways\n\n- Expect premium, low-volume models first (2027-2028)\n- Mainstream price parity is a 2030s story\n- Semi-solid and improved lithium-ion will dominate this decade"
