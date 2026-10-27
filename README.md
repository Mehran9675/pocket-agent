# Pocket Agent

A private AI workspace for **Android, iPhone and iPad**: chat with models running on the
device or in the cloud, hand multi-step tasks to agents, and run deep research that reads
dozens of sources and writes a cited report. Written in Go with
[Giode](https://github.com/mehran9675/giode) - one codebase, native rendering, no webview.

<p>
<img src="docs/screenshots/phone-home.png" width="200" alt="New chat on a phone">
<img src="docs/screenshots/phone-chat.png" width="200" alt="A reply with code and lists">
<img src="docs/screenshots/phone-models.png" width="200" alt="Model picker sheet">
<img src="docs/screenshots/phone-drawer.png" width="200" alt="Conversation drawer">
</p>
<p>
<img src="docs/screenshots/ipad-agent.png" width="410" alt="Agent run on iPad with the inspector">
<img src="docs/screenshots/ipad-research.png" width="410" alt="Deep research in progress on iPad">
</p>

## Features

**Models**

- **On-device inference** with llama.cpp (Metal on iPhone/iPad). On Android the engine is a
  small download (a few MB, picked for the phone's CPU) that the app checks for updates at
  every launch and installs automatically; on iOS it's built into the app. A curated catalog of GGUF
  models sized for phones (Qwen3, Llama 3.2, Gemma 3, Phi-4 mini, SmolLM2) with a fit rating
  for the device's RAM, resumable downloads, Hugging Face search, GGUF import, prompt-cache
  reuse between turns, and context / creativity / length / GPU settings.
- **Cloud providers** with your own keys: Anthropic (official Go SDK, adaptive thinking,
  server-side refusal fallback), OpenAI, Google Gemini, OpenRouter, Groq, Mistral, DeepSeek,
  xAI, Together, and any OpenAI-compatible endpoint (vLLM, LiteLLM, llama.cpp server).
- **Your own computer**: Ollama and LM Studio over the local network.
- Model lists fetched from each provider, favorites, per-conversation model switching.

**Chat**

- Streaming Markdown: headings, lists, task lists, tables, quotes, links, code blocks with
  copy; visible **thinking** for reasoning models (Claude, Gemini, DeepSeek, Qwen3 `<think>`).
- **Voice input** on Android (system speech recognition).
- **Attachments**: photos from the library or camera (Android) for vision models, and documents - PDF, Word,
  Excel, PowerPoint, HTML, Markdown, CSV, JSON, code - extracted to text on the device.
- **Tools** the model can call: web search, read web page, calculator, date/time, a JavaScript
  sandbox, reading long attachments, long-term memory. Per-tool permissions with inline
  *Allow once / Always allow / Deny* approval.
- **MCP servers** (Streamable HTTP) add remote tools to chats and agents.
- Regenerate, edit and resend, copy, share, export to Markdown, rename, pin, search, per-chat
  system prompt, automatic titles, token and time stats.

**Agents**

- Five built-in agents (Generalist, Researcher, Coder, Data Analyst, Writer) and an editor for
  your own: icon, color, instructions, tools, step limit, approval policy, starter prompts.
- Agents keep a live **plan checklist**, use tools autonomously and write deliverables to a
  per-conversation **workspace** you can open, copy and share.

**Deep research**

- Plans sub-questions, searches and reads sources in parallel, takes notes, reflects on gaps
  for follow-up rounds (Quick / Standard / Deep), then streams a report with inline citations
  and a source list. Live progress card shows each phase and query.

**Design**

- Responsive shell: drawer on phones, persistent sidebar on tablets, sidebar + **inspector**
  (plan, workspace files, sources) on wide iPads.
- Dark, light or system theme, six accent colors, three text sizes; Inter and JetBrains Mono.
- Everything stays on the device: conversations, files and memory live in the app's private
  storage, API keys in the Keychain / Android Keystore; messages go only to the provider you pick.
- Native touches: swipe a conversation to pin or delete it, long-press for its menu, haptics,
  a notification when a reply finishes in the background, and the screen kept awake while a
  model runs on the device.

## Quick start (desktop, no keys needed)

```sh
make dev   # go run -tags devmock . with an offline "Demo" provider and the Giode MCP server
```

Then *Connect a cloud model → Demo (offline) → Connect*. The demo provider streams scripted
replies, tool calls, agent runs and research so every screen can be tried without an API key
or network. Remove `-tags devmock` (`make run`) for the real app.

## Building for phones and tablets

```sh
go install gioui.org/cmd/gogio@latest
make android                 # build/pocket-agent.apk (arm64, Android 9+)
make engine-android          # build/engine/libpocketllama-android-*.so, the downloadable engine
make llama-ios && make ios LLAMA=1 SIGNKEY=<profile>   # universal iPhone + iPad app, engine built in
```

Each GitHub release attaches the APK, the engine builds and `engine-manifest.json`, which
installed apps read at launch to download the engine and its updates.

See [docs/building.md](docs/building.md) for prerequisites, signing and troubleshooting.

## Project layout

```
main.go                 app entry: data dir, fonts, theme, window
assets/                 fonts (Inter, JetBrains Mono), Lucide icons, app icon
internal/llm            provider-neutral messages + Anthropic, OpenAI-compatible, Gemini adapters
engine/                 the on-device engine: a C interface over llama.cpp, built as a shared library
internal/local          engine loader and updater, model catalog, downloader
internal/agent          tool loop, history building, system prompt, deep research pipeline
internal/tools          built-in tools, web search engines, JavaScript sandbox, MCP client
internal/docs           attachment extraction (PDF, Office, HTML, images)
internal/store          data model and JSON persistence
internal/ui             screens - one view per file
internal/ui/widgets     component kit - one component per file
internal/ui/theme       design tokens, fonts, icons
scripts/                engine builds (build-engine.sh, build-llama.sh for iOS), manifest, app icon
docs/                   building, architecture
```

More in [docs/architecture.md](docs/architecture.md).


## Tests

```sh
make test   # providers (SSE parsing), tools, agent loop, history, store, engine updater, documents
make vet
```

## Third-party

Inter and JetBrains Mono (SIL Open Font License), Lucide icons (ISC), llama.cpp (MIT, in the
engine library), goldmark, goja, ledongthuc/pdf, the Anthropic Go SDK. License files
for the bundled fonts and icons are in `assets/`.
