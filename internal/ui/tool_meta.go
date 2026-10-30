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
		return "globe", verb(tr("Searching “{query}”", "query", s("query")), tr("Searched “{query}”", "query", s("query")))
	case "fetch_url":
		host := s("url")
		if u, err := url.Parse(host); err == nil && u.Host != "" {
			host = strings.TrimPrefix(u.Host, "www.")
		}
		return "link", verb(tr("Reading {source}", "source", host), tr("Read {source}", "source", host))
	case "calculator":
		return "calculator", verb(tr("Calculating {expression}", "expression", s("expression")), tr("Calculated {expression}", "expression", s("expression")))
	case "current_datetime":
		return "clock", verb(tr("Checking the time"), tr("Checked the time"))
	case "run_javascript":
		return "square-terminal", verb(tr("Running code"), tr("Ran code"))
	case "read_attachment":
		return "paperclip", verb(tr("Reading {source}", "source", s("name")), tr("Read {source}", "source", s("name")))
	case "list_files":
		return "folder", verb(tr("Listing workspace files"), tr("Listed workspace files"))
	case "read_file":
		return "file-text", verb(tr("Reading {source}", "source", s("path")), tr("Read {source}", "source", s("path")))
	case "write_file":
		return "pen-line", verb(tr("Writing {file}", "file", s("path")), tr("Wrote {file}", "file", s("path")))
	case "edit_file":
		return "pencil", verb(tr("Editing {file}", "file", s("path")), tr("Edited {file}", "file", s("path")))
	case "delete_file":
		return "trash-2", verb(tr("Deleting {file}", "file", s("path")), tr("Deleted {file}", "file", s("path")))
	case "update_plan":
		return "list-checks", tr("Updated the plan")
	case "remember":
		return "brain", verb(tr("Saving to memory"), tr("Saved to memory"))
	case "forget":
		return "brain", verb(tr("Forgetting"), tr("Removed from memory"))
	case "use_skill":
		return "book-open", verb(tr("Loading the {skill} skill", "skill", s("name")), tr("Loaded the {skill} skill", "skill", s("name")))
	case "read_skill_file":
		return "book-open", verb(tr("Reading {source}", "source", s("name")+"/"+s("path")), tr("Read {source}", "source", s("name")+"/"+s("path")))
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
	return "plug", verb(tr("Using {tool}", "tool", title), tr("Used {tool}", "tool", title))
}
