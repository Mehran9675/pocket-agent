package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"mime"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
)

func (a *App) agentByID(id string) *store.Agent {
	for _, ag := range a.agents {
		if ag.ID == id {
			return ag
		}
	}
	return nil
}

func (a *App) isRunning(c *store.Conversation) bool {
	if c == nil {
		return false
	}
	_, ok := a.runs[c.ID]
	return ok
}

func (a *App) copyText(s string) {
	a.win.SetClipboard(s)
	a.toast(giode.ToastSuccess, "Copied")
}

// workspaceFiles lists a conversation's workspace.
func (a *App) workspaceFiles(c *store.Conversation) []string {
	if c == nil {
		return nil
	}
	dir := filepath.Join(a.st.Dir(), "workspaces", c.ID)
	var out []string
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out
}

// dp converts a window size from pixels to dp.
func dpSize(px image.Point) image.Point {
	return image.Pt(int(giode.PxToDp(px.X)), int(giode.PxToDp(px.Y)))
}

func greeting() string {
	h := time.Now().Hour()
	switch {
	case h < 5:
		return "Working late?"
	case h < 12:
		return "Good morning"
	case h < 18:
		return "Good afternoon"
	}
	return "Good evening"
}

func compactNum(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 10_000:
		return fmt.Sprintf("%dk", n/1000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprint(n)
}

func ifStr(c bool, a, b string) string {
	if c {
		return a
	}
	return b
}

func titleFromText(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 48 {
		s = string(r[:48]) + "…"
	}
	if s == "" {
		s = "New chat"
	}
	return s
}

func lastPathElem(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		p = p[i+1:]
	}
	return p
}

func safeFileName(s string) string {
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) {
			return '-'
		}
		return r
	}, s)
	if s == "" {
		s = "conversation"
	}
	return s
}

func contextBG() context.Context { return context.Background() }

func fmtDuration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
}

func prettyJSON(s string) string {
	var v any
	if json.Unmarshal([]byte(s), &v) != nil {
		return s
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return s
	}
	return string(b)
}

func fileIcon(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".txt", ".rtf":
		return "file-text"
	case ".js", ".ts", ".py", ".go", ".html", ".css", ".json", ".sh", ".java", ".kt", ".swift", ".rs", ".c", ".cpp":
		return "file-code"
	case ".csv", ".tsv":
		return "database"
	}
	return "file"
}

func mimeOf(name string) string {
	if t := mime.TypeByExtension(filepath.Ext(name)); t != "" {
		return t
	}
	return "text/plain"
}

func isMobile() bool { return runtime.GOOS == "android" || runtime.GOOS == "ios" }

func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}

func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i > 0 {
		return s[:i+1]
	}
	return s
}

func dayBucket(t time.Time) string {
	now := time.Now()
	y1, m1, d1 := now.Date()
	today := time.Date(y1, m1, d1, 0, 0, 0, 0, now.Location())
	switch {
	case !t.Before(today):
		return "Today"
	case !t.Before(today.AddDate(0, 0, -1)):
		return "Yesterday"
	case !t.Before(today.AddDate(0, 0, -7)):
		return "Previous 7 days"
	case !t.Before(today.AddDate(0, 0, -30)):
		return "Previous 30 days"
	}
	return "Older"
}
