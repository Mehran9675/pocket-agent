package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// onboarding guides a first launch to a model.
func (v *chatView) onboarding(compact bool) widgets.El {
	a := v.a
	option := func(key, icon, color, title, body, cta string, onClick func()) widgets.El {
		return widgets.Tap(key, title, func(s giode.PressState) widgets.El {
			return widgets.Col(widgets.St{
				Gap: 10, Padding: giode.UniformInset(18), BorderRadius: 18, Width: -1,
				Background: widgets.HoverBg(s, theme.T.Surface), BorderWidth: 1, BorderColor: theme.T.Border,
			},
				widgets.Row(widgets.St{Gap: 12}, widgets.IconTile(icon, color, 40), widgets.Text(title, theme.SizeHeadline, theme.T.Text, giode.FontWeightBold)),
				widgets.Text(body, theme.SizeCallout, theme.T.Text2),
				widgets.Row(widgets.St{Gap: 6}, widgets.Text(cta, theme.SizeCallout, theme.T.PrimaryText, giode.FontWeightMedium), widgets.Icon("chevron-right", 16, theme.T.PrimaryText)),
			)
		}, onClick)
	}
	return giode.Box(widgets.St{Width: -1, Padding: giode.SymmetricInset(20, 24)}, widgets.Centered(560,
		widgets.Col(widgets.St{Width: -1, Gap: 16, Align: giode.AlignCenter},
			widgets.LogoMark(60),
			widgets.Text("Welcome to Pocket Agent", theme.SizeDisplay, theme.T.Text, giode.FontWeightBold),
			giode.Text("A private AI workspace in your pocket: chat, research and autonomous agents - running on your device or on the cloud models you choose.",
				widgets.St{FontSize: theme.SizeBody, Color: theme.T.Muted, TextAlign: giode.TextAlignCenter, LineHeight: 22}),
			widgets.Gap(0, 4),
			option("onb/cloud", "cloud", "#7c83ff", "Connect a cloud model",
				"Use Claude, GPT, Gemini, OpenRouter, Groq and more with your own API key. Keys stay on this device.",
				"Add a provider", func() { a.navigate("settings/providers") }),
			option("onb/local", "cpu", "#34d399", "Run a model on this device",
				"Download an open model like Qwen3, Llama 3.2 or Gemma 3. Works offline, nothing leaves your phone.",
				"Open the model library", func() { a.navigate("library") }),
			option("onb/lan", "server", "#f59e0b", "Use a computer on your network",
				"Point Pocket Agent at Ollama or LM Studio running on your Mac or PC.",
				"Set up a local server", func() { a.navigate("settings/provider/ollama") }),
		)))
}
