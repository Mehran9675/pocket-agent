package widgets

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/toggle"
)

// Toggle is an on/off switch kept under key, showing on and calling
// onChange when flipped.
func Toggle(key string, on bool, onChange func(bool)) El {
	s := giode.Keyed("switch", key, func() *toggle.Switch { return giode.Input.Switch("") })
	s.SetChecked(on)
	s.OnChange(onChange)
	return s
}
