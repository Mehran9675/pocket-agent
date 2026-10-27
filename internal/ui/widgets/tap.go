package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// Tap returns a pressable kept under key whose content and handler are
// refreshed every frame. The optional styles size the press area (its
// Width or FlexGrow) and are fixed when the key is first seen.
func Tap(key string, label string, content func(s giode.PressState) El, onClick func(), st ...St) El {
	p := giode.Keyed("tap", key, func() *giode.PressableElement {
		return giode.Pressable(content, st...)
	})
	p.Content(content).Label(label).OnClick(func(giode.ClickEvent) {
		if onClick != nil {
			onClick()
		}
	})
	return p
}

// TapAlt is Tap with a secondary action, run on a right click or a
// long press.
func TapAlt(key, label string, content func(s giode.PressState) El, onClick, onAlt func()) El {
	p := giode.Keyed("tap", key, func() *giode.PressableElement { return giode.Pressable(content) })
	p.Content(content).Label(label).OnClick(func(e giode.ClickEvent) {
		if e.Button == giode.MouseSecondary {
			return
		}
		if onClick != nil {
			onClick()
		}
	}).OnSecondaryClick(func(giode.ClickEvent) {
		if onAlt != nil {
			onAlt()
		}
	}).OnLongPress(func(giode.ClickEvent) {
		if onAlt != nil {
			giode.Haptic(giode.HapticMedium)
			onAlt()
		}
	})
	return p
}

func HoverBg(s giode.PressState, base string) string {
	switch {
	case s.Pressed:
		return theme.T.Pressed
	case s.Hovered:
		return theme.T.Hover
	}
	return base
}
