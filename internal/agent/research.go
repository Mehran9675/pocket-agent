package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

// ResearchObserver receives a research run's progress, on the run's
// goroutines (calls are serialized).
type ResearchObserver interface {
	// Update mutates the run's state; the UI applies it on its own
	// goroutine.
	Update(fn func(r *store.Research))
	// Text streams the report.
	Text(delta string)
	Usage(u llm.Usage)
}

// Research configures a deep research run.
type Research struct {
	Provider llm.Provider
	Model    string
	Question string
	// Context is earlier conversation the question builds on.
	Context  string
	Depth    store.ResearchDepth
	Search   tools.SearchConfig
	Small    bool // small context window: read less, write tighter
	// Thinking sets how much a reasoning model thinks on each call.
	Thinking llm.Thinking
	Observer ResearchObserver

	mu     sync.Mutex
	notes  []note
	seen   map[string]int // URL → source index
	usage  llm.Usage
	obsMu  sync.Mutex
	nextSI int
}

type note struct {
	source   int
	question string
	text     string
}

type plan struct {
	Goal    string `json:"goal"`
	Queries []struct {
		Question string `json:"question"`
		Query    string `json:"query"`
	} `json:"queries"`
	Done bool `json:"done"`
}

// depthSpec sizes a run: sub-questions planned, search rounds, pages
// read per query and follow-up queries per round. Reflection may end a
// run early only after minRounds and once targetSources pages were
// read.
type depthSpec struct {
	questions, rounds, minRounds, pages, followups, targetSources int
}

func (r *Research) spec() depthSpec {
	switch r.Depth {
	case store.DepthQuick:
		return depthSpec{questions: 4, rounds: 1, minRounds: 1, pages: 3, followups: 0, targetSources: 8}
	case store.DepthDeep:
		return depthSpec{questions: 8, rounds: 5, minRounds: 3, pages: 5, followups: 6, targetSources: 40}
	}
	return depthSpec{questions: 6, rounds: 3, minRounds: 2, pages: 4, followups: 4, targetSources: 20}
}

func (r *Research) update(fn func(*store.Research)) {
	r.obsMu.Lock()
	defer r.obsMu.Unlock()
	r.Observer.Update(fn)
}

func (r *Research) complete(ctx context.Context, system, prompt string, maxTokens int) (string, error) {
	resp, err := chat(ctx, r.Provider, llm.Request{
		Model:     r.Model,
		System:    system,
		Messages:  []llm.Message{{Role: llm.RoleUser, Parts: []llm.Part{llm.TextPart(prompt)}}},
		MaxTokens: maxTokens,
		Thinking:  r.Thinking,
	}, nil)
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	r.usage = r.usage.Add(resp.Usage)
	r.mu.Unlock()
	if resp.StopReason == "refusal" {
		return "", ErrRefused
	}
	return resp.Text, nil
}

var jsonObject = regexp.MustCompile(`(?s)\{.*\}`)

func parsePlan(s string) (plan, error) {
	var p plan
	m := jsonObject.FindString(s)
	if m == "" {
		return p, errors.New("the model didn't return a research plan")
	}
	if err := json.Unmarshal([]byte(m), &p); err != nil {
		return p, fmt.Errorf("couldn't read the research plan: %w", err)
	}
	return p, nil
}

// Run executes the research and streams the report.
func (r *Research) Run(ctx context.Context) error {
	r.seen = map[string]int{}
	sp := r.spec()
	t0 := time.Now()
	defer func() {
		r.Observer.Usage(r.usage)
		r.update(func(s *store.Research) { s.Elapsed = time.Since(t0) })
	}()
	r.update(func(s *store.Research) {
		s.Phase, s.Started, s.Rounds = store.PhasePlanning, t0, sp.rounds
	})

	today := time.Now().Format("2 January 2006")
	ctxNote := ""
	if r.Context != "" {
		ctxNote = "\n\nEarlier conversation, for context:\n" + r.Context
	}
	raw, err := r.complete(ctx, "You plan web research. Today is "+today+". Reply with JSON only.",
		fmt.Sprintf(`Research request: %s%s

Break it into %d focused sub-questions that together answer it thoroughly, each with a web search query (concise, keyword style, include years for time-sensitive topics).
Reply with JSON: {"goal": "one sentence restating what the final report must answer", "queries": [{"question": "...", "query": "..."}]}`,
			r.Question, ctxNote, sp.questions), 1500)
	if err != nil {
		return r.fail(err)
	}
	p, err := parsePlan(raw)
	if err != nil || len(p.Queries) == 0 {
		// Fall back to searching the question itself.
		p = plan{Goal: r.Question}
		p.Queries = append(p.Queries, struct {
			Question string `json:"question"`
			Query    string `json:"query"`
		}{r.Question, r.Question})
	}
	r.update(func(s *store.Research) {
		s.Goal = p.Goal
		for _, q := range p.Queries {
			s.Queries = append(s.Queries, store.ResearchQuery{Question: q.Question, Query: q.Query, Status: "pending", Round: 1})
		}
	})

	pending := p.Queries
	for round := 1; round <= sp.rounds && len(pending) > 0; round++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rd := round
		r.update(func(s *store.Research) { s.Phase, s.Round = store.PhaseSearching, rd })
		var wg sync.WaitGroup
		sem := make(chan struct{}, 3)
		for _, q := range pending {
			wg.Add(1)
			sem <- struct{}{}
			go func(question, query string) {
				defer wg.Done()
				defer func() { <-sem }()
				r.investigate(ctx, question, query, sp.pages)
			}(q.Question, q.Query)
		}
		wg.Wait()
		pending = nil
		if round == sp.rounds || sp.followups == 0 || ctx.Err() != nil {
			break
		}
		r.update(func(s *store.Research) { s.Phase = store.PhaseReflecting })
		next, err := r.reflect(ctx, p.Goal, sp.followups)
		if err != nil {
			break
		}
		// The model may call it done only once the run is deep enough.
		r.mu.Lock()
		read := len(r.notes)
		r.mu.Unlock()
		if next.Done && round >= sp.minRounds && read >= sp.targetSources {
			break
		}
		pending = next.Queries
		r.update(func(s *store.Research) {
			for _, q := range next.Queries {
				s.Queries = append(s.Queries, store.ResearchQuery{Question: q.Question, Query: q.Query, Status: "pending", Round: rd + 1})
			}
		})
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.mu.Lock()
	n := len(r.notes)
	r.mu.Unlock()
	if n == 0 {
		return r.fail(errors.New("no usable sources were found - check web search in Settings, or rephrase the question"))
	}
	r.update(func(s *store.Research) { s.Phase = store.PhaseWriting })
	if err := r.write(ctx, p.Goal); err != nil {
		return r.fail(err)
	}
	r.update(func(s *store.Research) { s.Phase = store.PhaseDone })
	return nil
}

func (r *Research) fail(err error) error {
	if errors.Is(err, context.Canceled) {
		r.update(func(s *store.Research) { s.Phase = store.PhaseCanceledPart })
		return err
	}
	r.update(func(s *store.Research) { s.Phase = store.PhaseFailed })
	return err
}

func (r *Research) setQuery(query string, fn func(q *store.ResearchQuery)) {
	r.update(func(s *store.Research) {
		for i := range s.Queries {
			if s.Queries[i].Query == query {
				fn(&s.Queries[i])
			}
		}
	})
}

// investigate searches one query, reads the best new results and takes
// notes.
func (r *Research) investigate(ctx context.Context, question, query string, pages int) {
	r.setQuery(query, func(q *store.ResearchQuery) { q.Status = "searching" })
	results, err := tools.Search(ctx, r.Search, query, 10)
	if err != nil {
		r.setQuery(query, func(q *store.ResearchQuery) { q.Status = "failed" })
		return
	}
	r.setQuery(query, func(q *store.ResearchQuery) { q.Status, q.Found = "reading", len(results) })
	r.update(func(s *store.Research) {
		if s.Phase == store.PhaseSearching {
			s.Phase = store.PhaseReading
		}
	})
	var picked []tools.Source
	r.mu.Lock()
	for _, res := range results {
		if len(picked) >= pages {
			break
		}
		if _, dup := r.seen[res.URL]; dup || res.URL == "" {
			continue
		}
		r.seen[res.URL] = -1 // claimed
		picked = append(picked, res)
	}
	r.mu.Unlock()

	var wg sync.WaitGroup
	for _, src := range picked {
		wg.Add(1)
		go func(src tools.Source) {
			defer wg.Done()
			fctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			page, err := tools.Fetch(fctx, src.URL)
			cancel()
			text := src.Snippet
			title := src.Title
			if err == nil && len(strings.TrimSpace(page.Text)) > 200 {
				text = page.Text
				if page.Title != "" {
					title = page.Title
				}
			}
			if strings.TrimSpace(text) == "" {
				return
			}
			limit := 14000
			if r.Small {
				limit = 3500
			}
			if rr := []rune(text); len(rr) > limit {
				text = string(rr[:limit])
			}
			extracted, err := r.complete(ctx, "You extract research notes from web pages. Be faithful to the source; never add outside knowledge.",
				fmt.Sprintf("Question: %s\n\nSource: %s (%s)\n\n<page>\n%s\n</page>\n\n"+
					"Write the facts, figures, dates, names and short quotes from this page that help answer the question, as terse bullet points (at most 12). "+
					"Note the publication date if visible. If nothing is relevant, reply exactly NONE.", question, title, src.URL, text), 900)
			if err != nil || strings.TrimSpace(extracted) == "" || strings.HasPrefix(strings.TrimSpace(extracted), "NONE") {
				return
			}
			r.mu.Lock()
			idx := r.nextSI
			r.nextSI++
			r.seen[src.URL] = idx
			r.notes = append(r.notes, note{source: idx, question: question, text: strings.TrimSpace(extracted)})
			r.mu.Unlock()
			r.update(func(s *store.Research) {
				// Sources are appended in index order under the lock above,
				// but updates may interleave; place by index.
				for len(s.Sources) <= idx {
					s.Sources = append(s.Sources, tools.Source{})
				}
				s.Sources[idx] = tools.Source{Title: title, URL: src.URL}
				s.Notes++
			})
			r.setQuery(query, func(q *store.ResearchQuery) { q.Read++ })
		}(src)
	}
	wg.Wait()
	r.setQuery(query, func(q *store.ResearchQuery) { q.Status = "done" })
}

func (r *Research) notesText(maxPer int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b strings.Builder
	for _, n := range r.notes {
		t := n.text
		if rr := []rune(t); maxPer > 0 && len(rr) > maxPer {
			t = string(rr[:maxPer]) + "…"
		}
		fmt.Fprintf(&b, "[%d] (for: %s)\n%s\n\n", n.source+1, n.question, t)
	}
	return b.String()
}

func (r *Research) reflect(ctx context.Context, goal string, n int) (plan, error) {
	raw, err := r.complete(ctx, "You review research progress. Reply with JSON only.",
		fmt.Sprintf(`Goal: %s

Notes so far:
%s

Review these notes critically, as an expert would before publishing. What is still missing, thin, outdated, one-sided or contradictory? Think of primary sources and official data, recent developments, numbers that need confirming, expert or opposing views, and specifics (names, dates, figures) the report will need.
Propose up to %d new web searches that would fill those gaps; don't repeat covered ground. Always propose searches unless the notes are truly comprehensive.
Reply with JSON: {"done": true|false, "queries": [{"question": "...", "query": "..."}]} - "done": true only when the notes already answer every part of the goal in depth.`,
			goal, r.notesText(500), n), 1200)
	if err != nil {
		return plan{}, err
	}
	return parsePlan(raw)
}

func (r *Research) write(ctx context.Context, goal string) error {
	r.mu.Lock()
	var srcs strings.Builder
	urls := make([]string, r.nextSI)
	for u, i := range r.seen {
		if i >= 0 && i < len(urls) {
			urls[i] = u
		}
	}
	r.mu.Unlock()
	for i, u := range urls {
		fmt.Fprintf(&srcs, "[%d] %s\n", i+1, u)
	}
	length := "a thorough report (roughly 1500-2500 words)"
	maxTokens := 9000
	if r.Small {
		length, maxTokens = "a focused report (roughly 400 words)", 1500
	}
	if r.Depth == store.DepthDeep && !r.Small {
		length, maxTokens = "a comprehensive report (roughly 2500-4000 words)", 14000
	}
	prompt := fmt.Sprintf(`Research request: %s
Goal: %s

Sources:
%s
Research notes (numbered by source):
%s
Write %s in Markdown that answers the request:
- Start with a 2-3 sentence executive summary in bold-free plain prose.
- Then use ## headings for the main themes; use a table where comparing things.
- Cite every factual claim inline with the source number as a markdown link, e.g. [[3]](URL-of-source-3).
- Note disagreements between sources and how recent the information is.
- End with "## Key takeaways" (3-5 bullets). Do not add a sources list; the app shows it.
Use only the notes; say so where the evidence is thin.`, r.Question, goal, srcs.String(), r.notesText(0), length)
	resp, err := chat(ctx, r.Provider, llm.Request{
		Model:     r.Model,
		System:    "You are an expert research analyst writing for a smart, busy reader. Today is " + time.Now().Format("2 January 2006") + ".",
		Messages:  []llm.Message{{Role: llm.RoleUser, Parts: []llm.Part{llm.TextPart(prompt)}}},
		MaxTokens: maxTokens,
		Thinking:  r.Thinking,
	}, func(ev llm.Event) {
		if ev.Kind == llm.EventText {
			r.Observer.Text(ev.Text)
		}
	})
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.usage = r.usage.Add(resp.Usage)
	r.mu.Unlock()
	if resp.StopReason == "refusal" {
		return ErrRefused
	}
	return nil
}
