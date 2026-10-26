package ui

import (
	"github.com/mehran9675/pocket-agent/internal/store"
)

// answerApproval resolves a pending tool approval.
func (a *App) answerApproval(c *store.Conversation, callID, toolName string, allow, always bool) {
	rs, ok := a.runs[c.ID]
	if !ok {
		return
	}
	if always {
		a.settings.AutoApprove[toolName] = true
		a.saveSettings()
	}
	rs.mu.Lock()
	ch := rs.approvals[callID]
	delete(rs.approvals, callID)
	rs.mu.Unlock()
	if ch != nil {
		ch <- allow
	}
}
