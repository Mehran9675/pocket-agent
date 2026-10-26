# Architecture

```
            ┌───────────────────────── internal/ui (Giode) ─────────────────────────┐
            │ shell · sidebar · chat · composer · messages · sheets · settings ·    │
            │ library · agents · inspector         widgets/ markdown/ theme/        │
            └──────────────┬──────────────────────────────┬─────────────────────────┘
                 Post()    │ observer (ordered op queue)  │ store (JSON files)
            ┌──────────────▼───────────────┐      ┌───────▼────────┐
            │ internal/agent               │      │ internal/store │
            │  Run: model ↔ tools loop     │      └────────────────┘
            │  Research: plan→search→read  │
            │   →reflect→write             │
            └───────┬──────────────┬───────┘
                    │              │
        ┌───────────▼───┐   ┌──────▼─────────┐
        │ internal/llm  │   │ internal/tools │── internal/docs (PDF/Office/HTML)
        │ Anthropic SDK │   │ web · JS · fs  │── MCP servers (HTTP)
        │ OpenAI-compat │   │ plan · memory  │
        │ Gemini        │   └────────────────┘
        │ local.Provider│── internal/local ── llama.cpp (cgo, "llama" tag)
        └───────────────┘
```

## Layers

**`internal/llm`** defines one provider-neutral message format (`Message` with text/image
`Part`s, `ToolCall`s and tool results) and the `Provider` interface: `Chat` streams `Event`s
(text, reasoning) and returns the assembled `Response`; `Models` lists models. Adapters:

- `Anthropic` - the official Go SDK; adaptive thinking with summarized display when the user
  wants to see reasoning; eager tool-input streaming; the server-side refusal fallback on the
  models that support it; the SDK's native assistant message is kept in `Message.Native` so
  thinking blocks replay unchanged inside a tool loop.
- `OpenAI` - Chat Completions over SSE for OpenAI and every compatible server; accumulates
  streamed tool-call fragments; reads `reasoning_content`/`reasoning` deltas; splits
  `<think>` tags.
- `Gemini` - `streamGenerateContent` over SSE, function calling, thought parts, and the
  model's own parts (with thought signatures) replayed verbatim.

**`internal/local`** runs GGUF models: `Engine` (llama.cpp via cgo, a stub without the `llama`
tag) keeps one model loaded, reuses the longest common prefix of the KV cache between turns,
and cancels decoding through llama.cpp's abort callback. `Provider` adapts it to
`llm.Provider`, formats prompts with the model's own chat template (ChatML fallback), trims old
turns to fit the context, and offers tools through a `<tool_call>` prompt protocol parsed out
of the stream. `Store` manages the models directory, resumable downloads and Hugging Face
search.

**`internal/tools`** holds each tool as a JSON-schema'd function. Arguments are validated
(valid JSON, required keys) before running, so truncated streamed input becomes an error the
model can correct. Tools that change things outside the conversation set `Approval`. MCP
servers are adapted into the same `Tool` type.

**`internal/agent`** contains the loop every mode shares (`Run.Execute`): stream a reply, run
the requested tools - asking the user first where needed - append results, repeat until the
model answers without tools or the step limit is reached. `BuildHistory` turns stored
conversations back into model messages (assistant text and tool calls, then results).
`Research` is a separate pipeline: a JSON plan of sub-questions, parallel search and page
reading with per-source note extraction, optional reflection rounds for follow-up queries,
then a streamed report citing numbered sources.

**`internal/store`** is the data model (`Conversation` → `Message` → `Block`s of text,
reasoning or a `ToolRun`) persisted as JSON files in the app's private directory. Writes are
queued and coalesced on a background goroutine; `Flush` runs when the app goes to the
background.

**`internal/ui`** is the Giode interface. Screens are methods on `App`, one view per file;
reusable components are in `widgets` (one per file), design tokens and fonts in `theme`,
Markdown rendering in `markdown`.

## Threading

Giode runs layout and event handlers on one UI goroutine. All `App` state is owned by it.
Model calls, tools, downloads and file pickers run on their own goroutines and hand results
back with `App.Post`. A turn's streaming events go through an `observer`: events are queued in
arrival order (adjacent text deltas merged) and one scheduled flush applies them, so a delta
can never be applied ahead of an earlier tool call, and redraws stay cheap at high token rates.

Tool approval blocks the agent goroutine on a channel that the inline approval card answers.

## Rendering

Finished messages are wrapped in `giode.Memo`, keyed by message ID and rebuilt only when the
message's version, expanded sections or theme change; only the streaming reply is rebuilt each
frame. Parsed Markdown documents are cached per text block. The message list pins itself to the
bottom while content grows unless the user has scrolled up.

## Responsive layout

| Breakpoint | Width | Layout |
| --- | --- | --- |
| compact | < 700 dp | Full-screen pages, sidebar in a drawer, bottom sheets |
| medium | 700-1149 dp | Persistent sidebar, centered dialogs |
| expanded | ≥ 1150 dp | Sidebar + chat + inspector (plan, workspace, sources) for agent and research chats |

## Data location

`app.DataDir()/pocket-agent/` - `settings.json`, `agents.json`, `memory.json`,
`conversations/<id>.json`, `attachments/`, `workspaces/<conversation>/`, `models/*.gguf`.
