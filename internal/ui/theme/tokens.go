// Package theme holds Pocket Agent's design tokens - colors for light and
// dark mode, accents and typography - and registers its fonts and icons.
package theme

// Tokens are the app's semantic colors for the active theme. Views read
// them every frame, so a theme switch applies on the next frame.
type Tokens struct {
	Bg           string
	Sidebar      string
	Surface      string
	Surface2     string
	Elevated     string
	Hover        string
	Pressed      string
	Border       string
	BorderStrong string
	Text         string
	Text2        string
	Muted        string
	Faint        string
	Primary      string
	PrimarySoft  string
	PrimaryText  string
	OnPrimary    string
	Success      string
	SuccessSoft  string
	Warning      string
	WarningSoft  string
	Danger       string
	DangerSoft   string
	Info         string
	UserBubble   string
	CodeBg       string
	CodeBorder   string
	Scrim        string
	Shadow       string
	Dark         bool
}

// T holds the active tokens.
var T Tokens
