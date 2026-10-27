package ui

import (
	"context"
	"errors"
	"io"

	"github.com/mehran9675/giode"
)

func (a *App) importModel() {
	go func() {
		files, err := a.win.PickFiles(context.Background(), giode.PickOptions{Types: []string{"gguf"}})
		if errors.Is(err, giode.ErrCanceled) {
			return
		}
		if err != nil {
			a.toast(giode.ToastError, err.Error())
			return
		}
		if len(files) == 0 {
			return
		}
		f := files[0]
		defer f.Close()
		name := f.Name
		a.toast(giode.ToastInfo, "Importing "+name+"…")
		m, err := a.models.Import(name, io.Reader(f))
		if err != nil {
			a.toast(giode.ToastError, err.Error())
			return
		}
		a.toast(giode.ToastSuccess, "Imported "+m.Name)
	}()
}
