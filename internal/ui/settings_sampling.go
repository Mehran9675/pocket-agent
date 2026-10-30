package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/local"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

var repeatWindows = []int{0, 64, 128, 256, 512, 1024, -1}
var repeatWindowLabels = []string{key("Off"), "64", "128", "256", "512", "1024", key("Whole chat")}

// fmtOff formats v, or "Off" when it equals off.
func fmtOff(format string, off float64) func(float64) string {
	return func(v float64) string {
		if v == off {
			return tr("Off")
		}
		return fmt.Sprintf(format, v)
	}
}

// samplingBody tunes how on-device models pick each token.
func (a *App) samplingBody() widgets.El {
	ls := &a.settings.Local
	g := &ls.Sampling
	save := func() { a.saveSettings() }
	f32 := func(p *float32) func(float64) {
		return func(v float64) {
			*p = float32(v)
			save()
		}
	}
	slider := func(key, label, hint string, v *float32, lo, hi, step float64, format func(float64) string) widgets.El {
		return widgets.SliderRow("samp/"+key, label, hint, float64(*v), lo, hi, step, format, f32(v))
	}
	intSlider := func(key, label, hint string, v *int, lo, hi float64, format func(float64) string) widgets.El {
		return widgets.SliderRow("samp/"+key, label, hint, float64(*v), lo, hi, 1, format, func(x float64) {
			*v = int(x)
			save()
		})
	}

	seed := a.input("samp/seed", tr("Random"), "", func(i *input.Input) { i.Keyboard(giode.KeyboardNumeric) })
	if g.Seed != 0 && seed.Text() == "" {
		seed.SetText(strconv.FormatUint(uint64(g.Seed), 10))
	}
	if n, err := strconv.ParseUint(strings.TrimSpace(seed.Text()), 10, 32); err == nil && uint32(n) != g.Seed {
		g.Seed = uint32(n)
		save()
	} else if strings.TrimSpace(seed.Text()) == "" && g.Seed != 0 {
		g.Seed = 0
		save()
	}

	mirostat := g.Mirostat == 2
	truncation := []widgets.El{
		intSlider("topk", "Top-K", tr("Only the K likeliest tokens are considered."), &g.TopK, 0, 200, fmtOff("%.0f", 0)),
		slider("topp", "Top-P", tr("Only the likeliest tokens adding up to this probability."), &g.TopP, 0.5, 1, 0.01, fmtOff("%.2f", 1)),
		slider("minp", "Min-P", tr("Drops tokens less likely than this share of the top token's probability."), &g.MinP, 0, 0.5, 0.01, fmtOff("%.2f", 0)),
		slider("typical", "Typical-P", tr("Keeps tokens close to the expected surprise."), &g.TypicalP, 0.5, 1, 0.01, fmtOff("%.2f", 1)),
		slider("sigma", "Top-nσ", tr("Keeps tokens within n standard deviations of the top logit."), &g.TopNSigma, 0, 5, 0.1, fmtOff("%.1f", 0)),
		slider("xtcp", "XTC probability", tr("How often XTC removes the likeliest tokens, for less predictable writing."), &g.XTCProbability, 0, 1, 0.05, fmtOff("%.2f", 0)),
		slider("xtct", "XTC threshold", "", &g.XTCThreshold, 0, 0.5, 0.01, func(v float64) string { return fmt.Sprintf("%.2f", v) }),
	}
	if mirostat {
		truncation = []widgets.El{
			slider("tau", tr("Target entropy (τ)"), tr("Higher is more varied."), &g.MirostatTau, 0, 10, 0.1, func(v float64) string { return fmt.Sprintf("%.1f", v) }),
			slider("eta", tr("Learning rate (η)"), "", &g.MirostatEta, 0.01, 1, 0.01, func(v float64) string { return fmt.Sprintf("%.2f", v) }),
		}
	}

	return widgets.Col(widgets.St{Gap: 16, Width: -1},
		widgets.Text(tr("How on-device models pick each word. Changes apply to the next reply."), theme.SizeFootnote, theme.T.Muted),
		widgets.SectionLabel(tr("Basics")),
		widgets.Card(widgets.St{Gap: 18},
			widgets.SliderRow("samp/temp", tr("Temperature"), tr("Lower is focused and repeatable, higher more creative. 0 always picks the likeliest word."), ls.Temperature, 0, 2, 0.05,
				func(v float64) string { return fmt.Sprintf("%.2f", v) }, func(v float64) {
					ls.Temperature = v
					save()
				}),
			widgets.SliderRow("samp/max", tr("Maximum reply length"), "", float64(ls.MaxTokens), 256, 16384, 256,
				func(v float64) string { return fmt.Sprintf(tr("%.0f tokens"), v) }, func(v float64) {
					ls.MaxTokens = int(v)
					save()
				}),
			widgets.Row(widgets.St{Width: -1, Gap: 12},
				giode.Box(widgets.St{FlexGrow: 1}, widgets.Col(widgets.St{Gap: 2},
					widgets.Text(tr("Seed"), theme.SizeCallout, theme.T.Text, giode.FontWeightMedium),
					widgets.Text(tr("The same seed and settings give the same reply."), theme.SizeCaption, theme.T.Muted))),
				giode.Box(widgets.St{Width: 130}, seed)),
		),
		widgets.SectionLabel(tr("Choosing tokens")),
		widgets.Card(widgets.St{Gap: 18}, append([]widgets.El{
			widgets.SelectRow("samp/mirostat", "Mirostat", tr("Mirostat 2.0 steers toward a steady level of surprise and replaces the samplers below."), []string{tr("Off"), "Mirostat 2.0"}, ifInt(mirostat, 1, 0), func(i int) {
				g.Mirostat = ifInt(i == 1, 2, 0)
				save()
			}),
		}, truncation...)...),
		widgets.SectionLabel(tr("Repetition")),
		widgets.Card(widgets.St{Gap: 18},
			slider("rep", tr("Repeat penalty"), tr("Discourages reusing recent tokens."), &g.RepeatPenalty, 1, 2, 0.01, fmtOff("%.2f", 1)),
			slider("freq", tr("Frequency penalty"), tr("Grows with how often a token has appeared."), &g.FrequencyPenalty, 0, 2, 0.05, fmtOff("%.2f", 0)),
			slider("pres", tr("Presence penalty"), tr("Applies once a token has appeared at all."), &g.PresencePenalty, 0, 2, 0.05, fmtOff("%.2f", 0)),
			widgets.SelectRow("samp/window", tr("Penalty window"), tr("How many recent tokens the penalties look at."), trAll(repeatWindowLabels), indexOr(repeatWindows, g.RepeatLastN, 1), func(i int) {
				g.RepeatLastN = repeatWindows[i]
				save()
			}),
			slider("dry", tr("DRY multiplier"), tr("“Don't repeat yourself”: penalizes repeating whole phrases. 0.8 is a good start."), &g.DRYMultiplier, 0, 2, 0.05, fmtOff("%.2f", 0)),
			slider("drybase", "DRY base", "", &g.DRYBase, 1, 3, 0.05, func(v float64) string { return fmt.Sprintf("%.2f", v) }),
			intSlider("drylen", tr("DRY allowed length"), tr("Phrases this long may repeat freely."), &g.DRYAllowedLength, 1, 10, func(v float64) string { return fmt.Sprintf("%.0f", v) }),
		),
		widgets.Button("samp/reset", tr("Reset to defaults"), "rotate-ccw", widgets.BtnSecondary, func() {
			ls.Sampling = local.DefaultGenParams()
			ls.Temperature, ls.MaxTokens = 0.7, 2048
			seed.SetText("")
			save()
			a.toast(giode.ToastInfo, tr("Sampling reset"))
		}, widgets.St{Width: -1}),
	)
}
