package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// Backgrounded records that the app left the screen. Called from
// Config.OnBackground, which may run off the UI goroutine.
func (a *App) Backgrounded() { a.inBackground.Store(true) }

// Foregrounded records that the app is shown again.
func (a *App) Foregrounded() { a.inBackground.Store(false) }

// runStarted keeps the device awake for the first active run: the
// screen stays on while a model runs on the device, and the system is
// asked to let the run finish if the app leaves the screen.
func (a *App) runStarted(ref store.ModelRef) {
	if ref.Provider == store.LocalProviderID {
		a.localRuns++
		if a.localRuns == 1 {
			_ = a.win.KeepScreenOn(true)
		}
	}
	if len(a.runs) == 1 && a.releaseAwake == nil {
		if release, err := a.win.KeepAlive("Pocket Agent is finishing a reply"); err == nil {
			a.releaseAwake = release
		}
	}
}

// runEnded undoes runStarted and, when the app is in the background,
// tells the user the reply is ready.
func (a *App) runEnded(c *store.Conversation, msg *store.Message, ref store.ModelRef) {
	if ref.Provider == store.LocalProviderID {
		a.localRuns--
		if a.localRuns == 0 {
			_ = a.win.KeepScreenOn(false)
		}
	}
	if len(a.runs) == 0 && a.releaseAwake != nil {
		a.releaseAwake()
		a.releaseAwake = nil
	}
	if !a.inBackground.Load() || msg.Stopped {
		return
	}
	body := msg.Error
	if body == "" {
		body = strings.Join(strings.Fields(msg.ReplyText()), " ")
		if r := []rune(body); len(r) > 140 {
			body = string(r[:140]) + "…"
		}
	}
	title := c.Title
	if title == "" {
		title = "Reply ready"
	}
	go func() { _ = a.win.Notify(title, body) }()
}

// approvalNeeded nudges the user when a tool waits for permission.
func (a *App) approvalNeeded() {
	giode.Haptic(giode.HapticWarning)
	if a.inBackground.Load() {
		go func() { _ = a.win.Notify("Approval needed", "An agent is waiting for your permission to run a tool.") }()
	}
}
