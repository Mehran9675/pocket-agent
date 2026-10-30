package ui

import (
	"context"
	"errors"
	"runtime"

	"github.com/mehran9675/giode"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// dictationSupported reports whether speech input is available (Giode
// implements it on Android and iOS).
func dictationSupported() bool { return runtime.GOOS == "android" || runtime.GOOS == "ios" }

// micButton starts and stops dictation into the composer.
func (c *composer) micButton() widgets.El {
	on := c.dictation != nil
	color := theme.T.Text2
	if on {
		color = theme.T.Danger
	}
	return widgets.IconButton("composer/mic", "mic", tr("Dictate"), 20, color, func() {
		if on {
			c.stopDictation()
			return
		}
		c.startDictation()
	})
}

func (c *composer) startDictation() {
	ctx, cancel := context.WithCancel(context.Background())
	c.dictation = cancel
	prefix := c.in.Text()
	if prefix != "" {
		prefix += " "
	}
	giode.Haptic(giode.HapticLight)
	go func() {
		defer c.a.Post(func() { c.dictation = nil })
		results, err := c.a.win.Dictate(ctx, "")
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				c.a.toast(giode.ToastError, tr("Dictation unavailable: {error}", "error", err))
			}
			return
		}
		for r := range results {
			text := prefix + r.Text
			c.a.Post(func() { c.in.SetText(text) })
			if r.Final {
				return
			}
		}
	}()
}

func (c *composer) stopDictation() {
	if c.dictation != nil {
		c.dictation()
		c.dictation = nil
	}
}
