package ui

import (
	"context"
	"errors"
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

// pickFiles opens the system file picker and attaches the chosen files.
func (a *App) pickFiles() {
	go func() {
		files, err := a.win.PickFiles(context.Background(), giode.PickOptions{Multiple: true})
		if errors.Is(err, giode.ErrCanceled) {
			return
		}
		if err != nil {
			a.toast(giode.ToastError, tr("Couldn't open the picker: {error}", "error", err))
			return
		}
		for _, f := range files {
			a.attachPicked(f)
		}
	}()
}

// pickPhoto attaches a photo from the library or, on Android, the camera.
func (a *App) pickPhoto(source giode.PhotoSource) {
	go func() {
		f, err := a.win.PickPhoto(context.Background(), giode.PhotoOptions{Source: source, MaxSide: docs.MaxImageSide})
		if errors.Is(err, giode.ErrCanceled) {
			return
		}
		if err != nil {
			a.toast(giode.ToastError, tr("Couldn't get the photo: {error}", "error", err))
			return
		}
		a.attachPicked(f)
	}()
}

// attachPicked reads and imports one picked file.
func (a *App) attachPicked(f giode.PickedFile) {
	name := f.Name
	if name == "" {
		name = tr("Attachment")
	}
	a.Post(func() {
		pending := a.composer.addPending(name)
		go func() {
			defer f.Close()
			data, err := io.ReadAll(io.LimitReader(f, 64<<20))
			var att store.Attachment
			if err == nil {
				att, err = a.importAttachment(name, data)
			}
			a.Post(func() { a.composer.resolvePending(pending, att, err) })
		}()
	})
}
