package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// conversationMarkdown exports c.
func conversationMarkdown(c *store.Conversation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n_Exported from Pocket Agent on %s_\n\n", c.Title, time.Now().Format("2 Jan 2006 15:04"))
	for _, m := range c.Messages {
		switch m.Role {
		case llm.RoleUser:
			b.WriteString("## You\n\n" + m.Text + "\n\n")
			for _, at := range m.Attachments {
				fmt.Fprintf(&b, "_Attachment: %s_\n\n", at.Name)
			}
		case llm.RoleAssistant:
			b.WriteString("## Assistant")
			if m.Model != "" {
				b.WriteString(" (" + m.Model + ")")
			}
			b.WriteString("\n\n")
			for _, bl := range m.Blocks {
				switch bl.Kind {
				case store.BlockText:
					b.WriteString(bl.Text + "\n\n")
				case store.BlockTool:
					fmt.Fprintf(&b, "> 🔧 %s %s\n\n", bl.Tool.Name, bl.Tool.Args)
				}
			}
			if srcs := m.Sources(); len(srcs) > 0 {
				b.WriteString("**Sources**\n\n")
				for i, s := range srcs {
					fmt.Fprintf(&b, "%d. [%s](%s)\n", i+1, s.Title, s.URL)
				}
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

// exportConversation saves c as Markdown through the system picker.
func (a *App) exportConversation(c *store.Conversation) {
	md := conversationMarkdown(c)
	go func() {
		w, err := a.win.SaveFile(context.Background(), safeFileName(c.Title)+".md")
		if errors.Is(err, giode.ErrCanceled) {
			return
		}
		if err != nil {
			a.toast(giode.ToastError, "Export failed: "+err.Error())
			return
		}
		_, err = io.WriteString(w, md)
		w.Close()
		if err == nil {
			a.toast(giode.ToastSuccess, "Conversation exported")
		}
	}()
}

func (a *App) shareText(text, subject string) {
	go func() {
		err := a.win.Share(context.Background(), giode.ShareText{Text: text, Subject: subject})
		if errors.Is(err, giode.ErrNotSupported) {
			a.win.SetClipboard(text)
			a.toast(giode.ToastInfo, "Copied to the clipboard")
		} else if err == nil && !isMobile() {
			a.toast(giode.ToastInfo, "Copied to the clipboard")
		}
	}()
}

func (a *App) shareFile(path, mime string) {
	go func() {
		err := a.win.Share(context.Background(), giode.ShareFile{Path: path, MIME: mime})
		if err != nil {
			a.toast(giode.ToastError, "Couldn't share: "+err.Error())
		}
	}()
}

func (a *App) exportAll() {
	var b strings.Builder
	for _, c := range a.convs {
		b.WriteString(conversationMarkdown(c))
		b.WriteString("\n---\n\n")
	}
	text := b.String()
	go func() {
		w, err := a.win.SaveFile(contextBG(), "pocket-agent-export.md")
		if err != nil {
			return
		}
		_, _ = w.Write([]byte(text))
		w.Close()
		a.toast(giode.ToastSuccess, "Export saved")
	}()
}
