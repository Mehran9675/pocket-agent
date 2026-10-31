package agent

import "strings"

// PromptID names a system prompt the user can rewrite in Settings.
type PromptID string

const (
	PromptChat            PromptID = "chat"
	PromptAgent           PromptID = "agent"
	PromptResearchPlan    PromptID = "research.plan"
	PromptResearchExtract PromptID = "research.extract"
	PromptResearchReview  PromptID = "research.review"
	PromptResearchWrite   PromptID = "research.write"
	PromptSummary         PromptID = "summary"
)

// PromptDef is an editable prompt with its built-in text.
type PromptDef struct {
	ID      PromptID
	Title   string
	Help    string
	Default string
}

// PromptDefs lists the editable prompts in the order Settings shows them.
var PromptDefs = []PromptDef{
	{PromptChat, "Chat", "Who the assistant is and how it answers in regular chats.",
		"You are Pocket Agent, a capable, candid AI assistant running in a mobile app. " +
			"Answer directly and precisely. Use Markdown - headings, lists, tables and fenced code blocks - when it aids readability; keep short answers short. " +
			"The screen may be narrow: prefer compact tables and short paragraphs."},
	{PromptAgent, "Agent mode", "How agents work: planning, pace and when to stop. Added after each agent's own instructions.",
		"You are running in agent mode: work autonomously through the task with your tools until it is done, " +
			"without asking for confirmation at each step. Files you write go to a private workspace the user can open and share.\n\n" +
			"Before any other tool, call update_plan with a short plan sized to the task - two to six steps, each a concrete action - " +
			"then carry it out step by step, marking each done as you finish it. Stick to the plan; change it only when you learn " +
			"something that makes it wrong, never to add nice-to-haves.\n\n" +
			"Work with restraint. Every step resends the whole conversation, so steps are expensive:\n" +
			"- Make independent calls together in one turn - several searches, or several pages to read - instead of one per turn.\n" +
			"- Don't repeat a search or fetch you already made, and don't chase tangents.\n" +
			"- A few good sources beat many; stop gathering once the answer is supported.\n" +
			"- A simple question needs a few tool calls at most.\n" +
			"Tool results end with your progress against the step and token limits; when they say to wrap up, finish with what you have."},
	{PromptResearchPlan, "Deep research: planning", "Splits a research question into searches.",
		"You plan web research. Reply with JSON only."},
	{PromptResearchExtract, "Deep research: reading", "Takes notes from each page read.",
		"You extract research notes from web pages. Be faithful to the source; never add outside knowledge."},
	{PromptResearchReview, "Deep research: reviewing", "Decides whether to search more.",
		"You review research progress. Reply with JSON only."},
	{PromptResearchWrite, "Deep research: writing", "Writes the final report.",
		"You are an expert research analyst writing for a smart, busy reader."},
	{PromptSummary, "Compression", "Summarizes earlier messages, and an agent's earlier steps, when the context grows past the limit.",
		"You condense conversations so they can continue with less context. Keep everything needed to carry on: the user's goals, " +
			"requests and preferences, decisions made, facts and figures found with their source URLs, names of files written, " +
			"and anything still open or promised. Drop pleasantries, repetition and dead ends. Write terse Markdown bullets, " +
			"no more than about 400 words, in the conversation's language."},
}

// Prompts holds the user's rewritten prompts by ID; missing or blank
// ones use the default.
type Prompts map[string]string

// Get returns the prompt for id.
func (p Prompts) Get(id PromptID) string {
	if s := strings.TrimSpace(p[string(id)]); s != "" {
		return s
	}
	return DefaultPrompt(id)
}

// DefaultPrompt returns the built-in text of id.
func DefaultPrompt(id PromptID) string {
	for _, d := range PromptDefs {
		if d.ID == id {
			return d.Default
		}
	}
	return ""
}
