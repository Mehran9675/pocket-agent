package theme

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/store"
)

// SystemPrefersDark asks the platform for its color scheme, defaulting
// to dark when it can't tell.
func SystemPrefersDark() bool {
	dark, err := giode.SystemDarkMode()
	if err != nil {
		return true
	}
	return dark
}

// ApplyTheme sets the tokens and giode's palette for theme and accent.
func ApplyTheme(th store.Theme, accentID string) {
	a := accentByID(accentID)
	dark := th == store.ThemeDark || (th == "system" && SystemPrefersDark())
	if dark {
		T = Tokens{
			Bg: "#0c0d10", Sidebar: "#101115", Surface: "#15171c", Surface2: "#1b1e24", Elevated: "#1a1c22",
			Hover: "#ffffff0d", Pressed: "#ffffff17", Border: "#24272e", BorderStrong: "#33373f",
			Text: "#eceef2", Text2: "#b4b9c3", Muted: "#7d8390", Faint: "#565b66",
			Primary: a.Dark, PrimarySoft: a.Dark + "1f", PrimaryText: a.Dark, OnPrimary: a.OnDark,
			Success: "#3ccf91", SuccessSoft: "#3ccf911f", Warning: "#f5b544", WarningSoft: "#f5b5441f",
			Danger: "#f87171", DangerSoft: "#f871711c", Info: "#60a5fa",
			UserBubble: "#1d2027", CodeBg: "#111317", CodeBorder: "#23262d",
			Scrim: "#000000a6", Shadow: "#00000080", Dark: true,
		}
		T.Hover, T.Pressed = Mix("#ffffff", T.Sidebar, 0.055), Mix("#ffffff", T.Sidebar, 0.09)
	} else {
		T = Tokens{
			Bg: "#ffffff", Sidebar: "#f7f7f8", Surface: "#f4f5f7", Surface2: "#eef0f3", Elevated: "#ffffff",
			Hover: "#0f11150a", Pressed: "#0f111514", Border: "#e4e6ea", BorderStrong: "#d3d6dc",
			Text: "#111318", Text2: "#3f4551", Muted: "#6e7480", Faint: "#a2a7b0",
			Primary: a.Light, PrimarySoft: a.Light + "14", PrimaryText: a.Light, OnPrimary: a.OnLight,
			Success: "#16a34a", SuccessSoft: "#16a34a14", Warning: "#c2740c", WarningSoft: "#c2740c14",
			Danger: "#dc2626", DangerSoft: "#dc262612", Info: "#2563eb",
			UserBubble: "#f0f1f4", CodeBg: "#f6f7f9", CodeBorder: "#e4e6ea",
			Scrim: "#0f111566", Shadow: "#1018281f", Dark: false,
		}
		T.Hover, T.Pressed = Mix("#0f1115", T.Sidebar, 0.05), Mix("#0f1115", T.Sidebar, 0.09)
	}
	T.PrimarySoft = Mix(T.Primary, T.Surface, 0.14)
	T.SuccessSoft = Mix(T.Success, T.Surface, 0.12)
	T.WarningSoft = Mix(T.Warning, T.Surface, 0.10)
	T.DangerSoft = Mix(T.Danger, T.Surface, 0.10)
	p := giode.Palette{
		Primary: T.Primary, Secondary: T.Text2, Background: T.Bg, Surface: T.Surface2, Text: T.Text,
		Border: T.Border, OnPrimary: T.OnPrimary, Muted: T.Muted, Accent: T.Primary,
		Success: T.Success, Warning: T.Warning, Danger: T.Danger, Info: T.Info,
	}
	mode := giode.ModeLight
	if dark {
		mode = giode.ModeDark
	}
	giode.SetPalette(mode, p)
	giode.SetMode(mode)
	giode.InvalidateCaches()
}
