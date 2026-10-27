package widgets

import (
	"math"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/slider"

	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

// SliderRow is a labeled slider kept under key, over [lo, hi] in steps
// of step, showing the value through format.
func SliderRow(key, label, hint string, value, lo, hi, step float64, format func(float64) string, onChange func(float64)) El {
	s := giode.Keyed("slider", key, func() *slider.Slider {
		return giode.Input.Slider(St{Color: theme.T.Primary, Background: theme.T.Surface2, BorderColor: theme.T.Primary})
	})
	if !s.Dragging() {
		s.SetValue(float32((math.Max(lo, math.Min(hi, value)) - lo) / (hi - lo)))
	}
	s.OnChange(func(v float32) {
		x := lo + float64(v)*(hi-lo)
		if step > 0 {
			x = math.Round(x/step) * step
		}
		onChange(x)
	})
	kids := []El{
		Row(St{Width: -1, Gap: 8},
			giode.Box(St{FlexGrow: 1}, Text(label, theme.SizeCallout, theme.T.Text, giode.FontWeightMedium)),
			Text(format(value), theme.SizeFootnote, theme.T.Text2, giode.FontWeightMedium)),
		giode.Box(St{Width: -1}, s),
	}
	if hint != "" {
		kids = append(kids, Text(hint, theme.SizeCaption, theme.T.Muted))
	}
	return Col(St{Gap: 6, Width: -1}, kids...)
}
