package widgets

import (
	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"
)

func Spacer() El { return giode.Box(St{FlexGrow: 1}) }

func Gap(w, h int) El { return giode.Box(St{Width: pr.Width(w), Height: pr.Height(h)}) }

func Row(st St, kids ...El) El {
	if st.Align == "" {
		st.Align = giode.AlignCenter
	}
	if st.Wrap == "" {
		st.Wrap = giode.WrapNoWrap
	}
	return giode.Row(st, kids...)
}

func Col(st St, kids ...El) El { return giode.Stack(st, kids...) }

// When returns el, or an empty box when cond is false.
func When(cond bool, el func() El) El {
	if !cond {
		return giode.Box(St{Display: giode.DisplayNone})
	}
	return el()
}
