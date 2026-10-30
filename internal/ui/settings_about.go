package ui

import (
	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

func (a *App) aboutBody() widgets.El {
	return widgets.Col(widgets.St{Gap: 16, Width: -1, Align: giode.AlignCenter},
		widgets.Gap(0, 8),
		widgets.LogoMark(64),
		widgets.Text("Pocket Agent", theme.SizeDisplay, theme.T.Text, giode.FontWeightSemiBold),
		widgets.Text(tr("Version {version}", "version", Version), theme.SizeFootnote, theme.T.Muted),
		giode.Text(tr("Private AI chat, research and agents for iPhone, iPad and Android - on-device or with the cloud models you choose."), widgets.St{FontSize: theme.SizeCallout, Color: theme.T.Text2, TextAlign: giode.TextAlignCenter}),
		widgets.Group(
			widgets.ListRow("about/giode", "layers", theme.T.Primary, tr("Built with Giode"), tr("Native Go UI on Gio"), nil, func() { a.openLink("https://github.com/mehran9675/giode") }),
			widgets.ListRow("about/llama", "cpu", "#34d399", "llama.cpp", tr("On-device inference (MIT)"), nil, func() { a.openLink("https://github.com/ggml-org/llama.cpp") }),
			widgets.ListRow("about/fonts", "pen-line", theme.T.Muted, "Inter & JetBrains Mono", tr("SIL Open Font License"), nil, nil),
			widgets.ListRow("about/icons", "star", theme.T.Muted, "Lucide icons", tr("ISC License"), nil, nil),
		),
		widgets.Group(
			widgets.ListRow("about/frames", "gauge", theme.T.Muted, tr("Show frame times"), tr("Build and render time and frame rate in the corner, from the next launch"),
				widgets.Toggle("about/frames/t", a.settings.FrameTimes, func(v bool) {
					a.settings.FrameTimes = v
					a.saveSettings()
					a.toast(giode.ToastInfo, ifStr(v, tr("Restart Pocket Agent to show frame times"), tr("Restart Pocket Agent to hide frame times")))
				}), nil),
		),
		widgets.Text(tr("Your conversations, keys and files stay on this device. Messages go only to the model provider you choose."), theme.SizeCaption, theme.T.Muted),
	)
}
