package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/docs"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
)

// attachmentTexts returns the documents attached to c.
func (a *App) attachmentTexts(c *store.Conversation) []tools.Attachment {
	var out []tools.Attachment
	for _, m := range c.Messages {
		for _, att := range m.Attachments {
			if att.Kind == "image" {
				continue
			}
			b, err := os.ReadFile(a.st.AttachmentTextPath(att.ID))
			if err == nil {
				out = append(out, tools.Attachment{Name: att.Name, Text: string(b)})
			}
		}
	}
	return out
}

// importAttachment extracts a picked file and stores it.
func (a *App) importAttachment(name string, data []byte) (store.Attachment, error) {
	res, err := docs.Extract(name, data)
	if err != nil {
		return store.Attachment{}, err
	}
	att := store.Attachment{ID: store.NewID(), Name: name, Kind: string(res.Kind), MIME: res.MIME, Size: int64(len(data)), Pages: res.Pages}
	switch res.Kind {
	case docs.KindImage:
		if err := os.WriteFile(a.st.AttachmentPath(att.ID), res.Data, 0o600); err != nil {
			return att, err
		}
		att.Size = int64(len(res.Data))
	default:
		if err := os.WriteFile(a.st.AttachmentTextPath(att.ID), []byte(res.Text), 0o600); err != nil {
			return att, err
		}
		att.Chars = len([]rune(res.Text))
	}
	return att, nil
}

// pickFiles opens the system picker and attaches the chosen files.
func (a *App) pickFiles(images bool) {
	go func() {
		var filter []giode.FileFilter
		if images {
			filter = append(filter, giode.FileFilter{Extensions: []string{"png", "jpg", "jpeg", "webp", "gif"}})
		}
		files, err := a.win.OpenFiles(context.Background(), filter...)
		if errors.Is(err, giode.ErrCanceled) {
			return
		}
		if err != nil {
			a.toast(giode.ToastError, "Couldn't open the picker: "+err.Error())
			return
		}
		for i, f := range files {
			name := fileName(f, i, images)
			pending := a.composer.addPending(name)
			go func(f io.ReadCloser, name string, pending *pendingAttachment) {
				defer f.Close()
				data, err := io.ReadAll(io.LimitReader(f, 64<<20))
				var att store.Attachment
				if err == nil {
					att, err = a.importAttachment(name, data)
				}
				a.Post(func() { a.composer.resolvePending(pending, att, err) })
			}(f, name, pending)
		}
	}()
}

// fileName finds a picked file's name; Giode's picker doesn't report
// it on every platform (see docs/giode-requests.md).
func fileName(f io.ReadCloser, i int, image bool) string {
	if n, ok := f.(interface{ Name() string }); ok {
		if base := lastPathElem(n.Name()); base != "" {
			return base
		}
	}
	if image {
		return fmt.Sprintf("Image %d", i+1)
	}
	return fmt.Sprintf("Document %d", i+1)
}
