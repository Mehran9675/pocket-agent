package ui

import (
	"context"
	"errors"
	"io"

	"github.com/mehran9675/giode"
)

func (a *App) importModel() {
	go func() {
		f, err := a.win.OpenFile(context.Background(), giode.FileFilter{Extensions: []string{"gguf"}})
		if errors.Is(err, giode.ErrCanceled) {
			return
		}
		if err != nil {
			a.toast(giode.ToastError, err.Error())
			return
		}
		defer f.Close()
		name := fileName(f, 0, false)
		a.toast(giode.ToastInfo, "Importing "+name+"…")
		m, err := a.models.Import(name, io.Reader(f))
		if err != nil {
			a.toast(giode.ToastError, err.Error())
			return
		}
		a.toast(giode.ToastSuccess, "Imported "+m.Name)
	}()
}
