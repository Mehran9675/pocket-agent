package ui

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/mehran9675/pocket-agent/internal/store"
)

// toolMeta returns the icon and a human description of a tool call.
func (a *App) toolMeta(t *store.ToolRun) (icon, title string) {
	var args map[string]any
	_ = json.Unmarshal([]byte(t.Args), &args)
	s := func(k string) string {
		v, _ := args[k].(string)
		return v
	}
	done := t.Status == store.ToolDone || t.Status == store.ToolError
	verb := func(running, finished string) string {
		if done {
			return finished
		}
		return running
	}
	switch t.Name {
	case "web_search":
		return "globe", verb("Searching", "Searched") + " “" + s("query") + "”"
	case "fetch_url":
		host := s("url")
		if u, err := url.Parse(host); err == nil && u.Host != "" {
			host = strings.TrimPrefix(u.Host, "www.")
		}
		return "link", verb("Reading", "Read") + " " + host
	case "calculator":
		return "calculator", verb("Calculating", "Calculated") + " " + s("expression")
	case "current_datetime":
		return "clock", verb("Checking", "Checked") + " the time"
	case "run_javascript":
		return "square-terminal", verb("Running code", "Ran code")
	case "read_attachment":
		return "paperclip", verb("Reading", "Read") + " " + s("name")
	case "list_files":
		return "folder", verb("Listing", "Listed") + " workspace files"
	case "read_file":
		return "file-text", verb("Reading", "Read") + " " + s("path")
	case "write_file":
		return "pen-line", verb("Writing", "Wrote") + " " + s("path")
	case "edit_file":
		return "pencil", verb("Editing", "Edited") + " " + s("path")
	case "delete_file":
		return "trash-2", verb("Deleting", "Deleted") + " " + s("path")
	case "update_plan":
		return "list-checks", "Updated the plan"
	case "remember":
		return "brain", verb("Saving to memory", "Saved to memory")
	case "forget":
		return "brain", verb("Forgetting", "Removed from memory")
	}
	title = t.Name
	for _, ts := range a.mcpTools {
		for _, tool := range ts {
			if tool.Name == t.Name {
				title = tool.Title
				if i := strings.Index(tool.Group, "· "); i >= 0 {
					title = tool.Group[i+len("· "):] + " · " + tool.Title
				}
			}
		}
	}
	return "plug", verb("Using ", "Used ") + title
}
