package ui

import (
	"github.com/mehran9675/giode/i18n"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

// tr translates UI text. The English text is the key, so English needs
// no message file; {name} placeholders are filled from name, value
// pairs, and %-verbs survive for fmt.
func tr(s string, args ...any) string { return i18n.T(s, args...) }

// key marks text kept untranslated in a package-level value so the
// extractor finds it; translate it with tr where it's shown.
func key(s string) string { return s }

// trAll translates each of labels.
func trAll(labels []string) []string {
	out := make([]string, len(labels))
	for i, l := range labels {
		out[i] = tr(l)
	}
	return out
}

// agentName is how an agent is shown: built-in ones in the app's
// language, unless the user renamed them.
func agentName(ag *store.Agent) string {
	if ag.Builtin {
		return tr(ag.Name)
	}
	return ag.Name
}

func agentDescription(ag *store.Agent) string {
	if ag.Builtin {
		return tr(ag.Description)
	}
	return ag.Description
}

func agentStarters(ag *store.Agent) []string {
	if ag.Builtin {
		return trAll(ag.Starters)
	}
	return ag.Starters
}

// toolSummary is a tool's description for people: its first sentence.
// Built-in tools' are translated; the full description stays English
// for models.
func toolSummary(t *tools.Tool) string { return tr(firstSentence(t.Description)) }

// trn translates a count phrase, picking the language's plural form
// for n; {n} is n. English forms are in locales/en.json.
func trn(s string, n int, args ...any) string { return i18n.N(s, n, args...) }
