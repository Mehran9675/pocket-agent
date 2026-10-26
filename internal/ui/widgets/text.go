package widgets

import (
	"gioui.org/unit"
	"github.com/mehran9675/giode"
	pr "github.com/mehran9675/giode/styles/properties"
)

// TextStyle is a text style shorthand.
func TextStyle(size float32, color string, weight ...pr.FontWeight) St {
	st := St{FontSize: unit.Sp(size), Color: color}
	if len(weight) > 0 {
		st.FontWeight = weight[0]
	}
	return st
}

// Text renders a text run.
func Text(s string, size float32, color string, weight ...pr.FontWeight) El {
	return giode.Text(s, TextStyle(size, color, weight...))
}

// Line renders a single ellipsized line.
func Line(s string, size float32, color string, weight ...pr.FontWeight) El {
	st := TextStyle(size, color, weight...)
	st.TextOverflow = giode.TextOverflowEllipsis
	st.MaxLines = 1
	return giode.Text(s, st)
}
