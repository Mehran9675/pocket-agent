package theme

import (
	"gioui.org/font"
	"gioui.org/font/opentype"
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/assets"
)

// RegisterFonts loads the embedded fonts. Inter SemiBold stands in for
// "bold": it reads crisper on small screens.
func RegisterFonts() error {
	type face struct {
		file   string
		family string
		weight font.Weight
		style  font.Style
	}
	faces := []face{
		{"Inter-Regular.ttf", FontUI, font.Normal, font.Regular},
		{"Inter-Italic.ttf", FontUI, font.Normal, font.Italic},
		{"Inter-Medium.ttf", FontUI, font.Medium, font.Regular},
		{"Inter-SemiBold.ttf", FontUI, font.Bold, font.Regular},
		{"Inter-Bold.ttf", FontUI, font.ExtraBold, font.Regular},
		{"JetBrainsMono-Regular.ttf", FontMono, font.Normal, font.Regular},
	}
	var coll []font.FontFace
	for _, f := range faces {
		data, err := assets.Fonts.ReadFile("fonts/" + f.file)
		if err != nil {
			return err
		}
		ff, err := opentype.Parse(data)
		if err != nil {
			return err
		}
		coll = append(coll, font.FontFace{
			Font: font.Font{Typeface: font.Typeface(f.family), Weight: f.weight, Style: f.style},
			Face: ff,
		})
	}
	giode.RegisterFonts(coll)
	return nil
}
