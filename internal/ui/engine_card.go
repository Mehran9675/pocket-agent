package ui

import (
	"context"
	"fmt"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// checkEngine looks for a new engine build at launch and installs it
// when an engine is already installed.
func (a *App) checkEngine() {
	go a.runtime.CheckAndUpdate(context.Background())
}

// installEngine downloads the engine.
func (a *App) installEngine() {
	go func() {
		if err := a.runtime.Install(context.Background()); err == nil && a.runtime.Ready() {
			giode.Haptic(giode.HapticSuccess)
			a.toast(giode.ToastSuccess, "On-device engine installed")
		}
	}()
}

// engineCard shows the engine's install and update state, or nil when
// it's ready and there's nothing to report.
func (a *App) engineCard() widgets.El {
	s := a.runtime.Status()
	note := func(icon, color, bg, text string, action widgets.El) widgets.El {
		if action == nil {
			action = giode.Box(widgets.St{})
		}
		return widgets.Row(widgets.St{Gap: 10, Padding: giode.UniformInset(12), BorderRadius: 12, Background: bg, Width: -1},
			widgets.Icon(icon, 18, color),
			giode.Box(widgets.St{FlexGrow: 1}, widgets.Text(text, theme.SizeFootnote, theme.T.Text)),
			action)
	}
	switch {
	case s.Builtin:
		return nil
	case !s.Supported:
		return note("triangle-alert", theme.T.Warning, theme.T.WarningSoft, a.runtime.Problem(), nil)
	case s.Downloading:
		title := "Downloading the on-device engine"
		if s.Installed != "" {
			title = "Updating the on-device engine"
		}
		var p float32
		if s.Size > 0 {
			p = float32(s.Done) / float32(s.Size)
		}
		return widgets.Card(widgets.St{Gap: 10},
			widgets.Row(widgets.St{Gap: 10, Width: -1},
				widgets.IconTile("cpu", "#34d399", 34),
				widgets.Col(widgets.St{Gap: 2, FlexGrow: 1},
					widgets.Text(title, theme.SizeCallout, theme.T.Text, giode.FontWeightSemiBold),
					widgets.Text(fmt.Sprintf("%s of %s", tools.HumanBytes(s.Done), tools.HumanBytes(s.Size)), theme.SizeFootnote, theme.T.Muted)),
				widgets.IconButton("engine/cancel", "x", "Cancel", 16, theme.T.Muted, a.runtime.Cancel)),
			giode.Progress(p, widgets.St{Color: "#34d399", Background: theme.T.Surface2, Height: 5}))
	case s.Installed == "":
		sub := "llama.cpp runs models privately on this " + deviceNoun() + ". It's a one-time download; updates install automatically."
		label := "Download engine"
		if s.Size > 0 {
			label = "Download engine · " + tools.HumanBytes(s.Size)
		}
		errRow := widgets.When(s.Err != "", func() widgets.El {
			return widgets.Text(s.Err, theme.SizeFootnote, theme.T.Danger)
		})
		return widgets.Card(widgets.St{Gap: 12},
			widgets.Row(widgets.St{Gap: 12, Width: -1},
				widgets.IconTile("cpu", "#34d399", 40),
				widgets.Col(widgets.St{Gap: 2, FlexGrow: 1},
					widgets.Text("Install the on-device engine", theme.SizeHeadline, theme.T.Text, giode.FontWeightSemiBold),
					widgets.Text(sub, theme.SizeFootnote, theme.T.Text2))),
			errRow,
			widgets.Button("engine/install", label, "download", widgets.BtnPrimary, a.installEngine, widgets.St{Width: -1}))
	case s.RestartPending():
		return note("refresh-cw", theme.T.Primary, theme.T.PrimarySoft, "The engine was updated to "+s.Installed+". Restart Pocket Agent to use it.", nil)
	case s.Err != "" && s.UpdateAvailable():
		return note("triangle-alert", theme.T.Warning, theme.T.WarningSoft, "Couldn't update the engine: "+s.Err,
			widgets.Button("engine/retry", "Retry", "", widgets.BtnSecondary, a.installEngine))
	}
	return nil
}

// engineLabel names the installed engine for summaries.
func (a *App) engineLabel() string {
	s := a.runtime.Status()
	switch {
	case s.Builtin:
		return "llama.cpp"
	case s.Installed != "":
		return "llama.cpp " + s.Installed
	case s.Supported:
		return "Not installed"
	}
	return "Unavailable"
}
