package ui

import (
	"context"
	"sync"
	"time"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

// observer applies a run's events to its message on the UI goroutine.
// Events queue in arrival order and one scheduled flush applies them,
// so text streamed after a tool call can never land before it; adjacent
// text deltas are merged to keep redraws cheap.
type observer struct {
	a  *App
	c  *store.Conversation
	m  *store.Message
	rs *runState

	mu        sync.Mutex
	ops       []observerOp
	scheduled bool
	lastSave  time.Time
}

// observerOp is a queued text delta (kind set) or UI change (fn set).
type observerOp struct {
	kind store.BlockKind
	text string
	fn   func()
}

func (o *observer) push(op observerOp) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if n := len(o.ops); op.fn == nil && n > 0 && o.ops[n-1].fn == nil && o.ops[n-1].kind == op.kind {
		o.ops[n-1].text += op.text
	} else {
		o.ops = append(o.ops, op)
	}
	if !o.scheduled {
		o.scheduled = true
		o.a.Post(o.flush)
	}
}

// post queues a change to the message.
func (o *observer) post(fn func()) { o.push(observerOp{fn: fn}) }

// flush applies the queued events; UI goroutine only.
func (o *observer) flush() {
	o.mu.Lock()
	ops := o.ops
	o.ops = nil
	o.scheduled = false
	o.mu.Unlock()
	for _, op := range ops {
		if op.fn != nil {
			op.fn()
		} else {
			o.appendBlock(op.kind, op.text)
		}
	}
	o.m.Version++
	o.c.Version++
}

func (o *observer) appendBlock(kind store.BlockKind, s string) {
	bl := o.m.Blocks
	if n := len(bl); n > 0 && bl[n-1].Kind == kind {
		bl[n-1].Text += s
		return
	}
	o.m.Blocks = append(o.m.Blocks, store.Block{Kind: kind, Text: s})
}

func (o *observer) StepStart(int) {}

func (o *observer) Text(delta string) { o.push(observerOp{kind: store.BlockText, text: delta}) }

func (o *observer) Reasoning(delta string) {
	o.push(observerOp{kind: store.BlockReasoning, text: delta})
}

func (o *observer) findTool(id string) *store.ToolRun {
	for i := len(o.m.Blocks) - 1; i >= 0; i-- {
		if t := o.m.Blocks[i].Tool; t != nil && t.CallID == id {
			return t
		}
	}
	return nil
}

func (o *observer) ToolCall(call llm.ToolCall, _ *tools.Tool) {
	o.post(func() {
		o.m.Blocks = append(o.m.Blocks, store.Block{Kind: store.BlockTool, Tool: &store.ToolRun{
			CallID: call.ID, Name: call.Name, Args: string(call.Args), Status: store.ToolRunning, Started: time.Now(),
		}})
	})
}

func (o *observer) ToolRunning(call llm.ToolCall) {
	o.post(func() {
		if t := o.findTool(call.ID); t != nil {
			t.Status, t.Started = store.ToolRunning, time.Now()
		}
	})
}

func (o *observer) ToolDenied(call llm.ToolCall) {
	o.post(func() {
		if t := o.findTool(call.ID); t != nil {
			t.Status = store.ToolDenied
		}
	})
}

func (o *observer) ToolResult(call llm.ToolCall, res tools.Result, d time.Duration) {
	o.post(func() {
		if t := o.findTool(call.ID); t != nil {
			t.Result, t.IsError, t.Duration = res.Text, res.IsError, d
			t.Sources, t.Files = res.Sources, res.Files
			t.Status = store.ToolDone
			if res.IsError {
				t.Status = store.ToolError
			}
		}
		o.a.st.SaveConversation(o.c)
	})
}

func (o *observer) Usage(u llm.Usage) {
	o.post(func() { o.m.Usage = o.m.Usage.Add(u) })
}

func (o *observer) plan(items []tools.PlanItem) {
	o.post(func() { o.m.Plan = items })
}

// approve asks the user about a tool call and waits for the answer.
func (o *observer) approve(ctx context.Context, call llm.ToolCall) bool {
	ch := make(chan bool, 1)
	o.rs.mu.Lock()
	o.rs.approvals[call.ID] = ch
	o.rs.mu.Unlock()
	o.post(func() {
		if t := o.findTool(call.ID); t != nil {
			t.Status = store.ToolPending
		}
	})
	select {
	case ok := <-ch:
		return ok
	case <-ctx.Done():
		return false
	}
}

// Update implements agent.ResearchObserver.
func (o *observer) Update(fn func(r *store.Research)) {
	o.post(func() {
		if o.m.Research == nil {
			o.m.Research = &store.Research{}
		}
		fn(o.m.Research)
		if time.Since(o.lastSave) > 3*time.Second {
			o.lastSave = time.Now()
			o.a.st.SaveConversation(o.c)
		}
	})
}
