package ui

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/pocket-agent/internal/docs"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// pendingAttachment is a file being imported into the composer.
type pendingAttachment struct {
	id   int
	name string
	att  store.Attachment
	err  error
	done bool
}

type composer struct {
	// dictation stops the running speech recognition, nil when idle.
	dictation func()
	a         *App
	in        *input.Input
	pending   []*pendingAttachment
	nextID    int
	thumbs    map[string]image.Image
}

func newComposer(a *App) *composer {
	c := &composer{a: a, thumbs: map[string]image.Image{}}
	// Grows from one to eight lines. With a hardware keyboard Enter
	// sends and Shift+Enter starts a new line; phones send with the
	// button.
	c.in = giode.Input.Text("Message Pocket Agent", widgets.St{
		Background: "transparent", BorderColor: "transparent", Padding: giode.Inset{Top: 6, Bottom: 6, Left: 4, Right: 4},
		Color: "theme.text", FontSize: theme.SizeBody,
	}).Lines(1, 8).SubmitOn(giode.SubmitEnterUnlessShift).Label("Message")
	c.in.Submit(func(string) { c.submit() })
	return c
}

func (c *composer) reset() {
	c.in.SetText("")
	c.pending = nil
}

func (c *composer) focus() {
	if !isMobile() {
		c.in.Focus()
	}
}

func (c *composer) load(text string, atts []store.Attachment) {
	c.in.SetText(text)
	c.pending = nil
	for _, at := range atts {
		c.nextID++
		c.pending = append(c.pending, &pendingAttachment{id: c.nextID, name: at.Name, att: at, done: true})
	}
	c.in.Focus()
}

func (c *composer) addPending(name string) *pendingAttachment {
	p := &pendingAttachment{name: name}
	c.a.Post(func() {
		c.nextID++
		p.id = c.nextID
		c.pending = append(c.pending, p)
	})
	return p
}

func (c *composer) resolvePending(p *pendingAttachment, att store.Attachment, err error) {
	p.done, p.att, p.err = true, att, err
	if err != nil {
		c.a.toast(giode.ToastError, p.name+": "+err.Error())
		c.remove(p)
	}
}

func (c *composer) remove(p *pendingAttachment) {
	for i, x := range c.pending {
		if x == p {
			c.pending = append(c.pending[:i], c.pending[i+1:]...)
			return
		}
	}
}

func (c *composer) busy() bool {
	for _, p := range c.pending {
		if !p.done {
			return true
		}
	}
	return false
}

func (c *composer) submit() {
	if c.busy() {
		return
	}
	var atts []store.Attachment
	for _, p := range c.pending {
		if p.err == nil {
			atts = append(atts, p.att)
		}
	}
	text := c.in.Text()
	if text == "" && len(atts) == 0 {
		return
	}
	c.stopDictation()
	c.reset()
	giode.Haptic(giode.HapticLight)
	if isMobile() {
		c.a.win.HideKeyboard()
	}
	c.a.send(text, atts)
}

func (c *composer) thumb(att store.Attachment) image.Image {
	if img, ok := c.thumbs[att.ID]; ok {
		return img
	}
	var img image.Image
	if f, err := os.Open(c.a.st.AttachmentPath(att.ID)); err == nil {
		if full, _, err := image.Decode(f); err == nil {
			img = docs.Thumbnail(full, 160)
		}
		f.Close()
	}
	c.thumbs[att.ID] = img
	return img
}

// view renders the composer.
func (c *composer) view(compact bool) widgets.El {
	a := c.a
	conv := a.current
	running := a.isRunning(conv)
	mode, agentID, web, depth := a.draft.mode, a.draft.agentID, a.draft.web, a.draft.depth
	if conv != nil {
		mode, agentID, web, depth = conv.Mode, conv.AgentID, conv.WebSearch, conv.Depth
	}

	var atts []widgets.El
	for _, p := range c.pending {
		p := p
		att := p.att
		if !p.done {
			att = store.Attachment{Name: p.name}
		}
		atts = append(atts, a.attachmentChip(fmt.Sprintf("pend/%d", p.id), att, !p.done, func() { c.remove(p) }))
	}

	placeholder := "Message Pocket Agent"
	switch mode {
	case store.ModeAgent:
		if ag := a.agentByID(agentID); ag != nil {
			placeholder = "Give " + ag.Name + " a task"
		}
	case store.ModeResearch:
		placeholder = "What should I research?"
	}

	// Mode chip
	modeLabel, modeIcon := "Chat", "message-square"
	switch mode {
	case store.ModeAgent:
		modeLabel, modeIcon = "Agent", "bot"
		if ag := a.agentByID(agentID); ag != nil {
			modeLabel, modeIcon = ag.Name, ag.Icon
		}
	case store.ModeResearch:
		modeLabel, modeIcon = "Deep research", "telescope"
	}
	tools := []widgets.El{
		widgets.IconButton("composer/attach", "plus", "Attach", 20, theme.T.Text2, a.openAttachSheet),
		widgets.Chip("composer/mode", modeLabel, modeIcon, mode != store.ModeChat, a.openModeSheet),
	}
	if mode == store.ModeChat {
		tools = append(tools, widgets.Chip("composer/web", ifStr(compact, "", "Web"), "globe", web, func() {
			if conv != nil {
				conv.WebSearch = !conv.WebSearch
				conv.Version++
				a.st.SaveConversation(conv)
			} else {
				a.draft.web = !a.draft.web
			}
		}))
	}
	if mode == store.ModeResearch {
		labels := map[store.ResearchDepth]string{store.DepthQuick: "Quick", store.DepthStandard: "Standard", store.DepthDeep: "Deep"}
		tools = append(tools, widgets.Chip("composer/depth", labels[depth], "gauge", false, a.openDepthSheet))
	}
	if a.modelThinks(a.currentModel()) {
		tools = append(tools, a.thinkingChip(compact))
	}
	canSend := !c.busy() && (c.in.Text() != "" || len(c.pending) > 0)
	if dictationSupported() && !running {
		tools = append(tools, c.micButton())
	}
	var action widgets.El
	if running {
		action = widgets.Tap("composer/stop", "Stop", func(s giode.PressState) widgets.El {
			return giode.Box(widgets.St{Width: 36, Height: 36, BorderRadius: 18, Background: theme.T.Text, Align: giode.AlignCenter, Justify: giode.JustifyCenter},
				giode.Box(widgets.St{Width: 12, Height: 12, BorderRadius: 3, Background: theme.T.Bg}))
		}, func() { a.stop(conv) })
	} else {
		action = widgets.Tap("composer/send", "Send", func(s giode.PressState) widgets.El {
			bg, fg := theme.T.Primary, theme.T.OnPrimary
			if !canSend {
				bg, fg = theme.T.Surface2, theme.T.Faint
			}
			st := widgets.St{Width: 36, Height: 36, BorderRadius: 18, Background: bg, Align: giode.AlignCenter, Justify: giode.JustifyCenter}
			if s.Pressed && canSend {
				st.Opacity = 0.8
			}
			return giode.Box(st, widgets.Icon("arrow-up", 20, fg))
		}, c.submit)
	}

	boxKids := []widgets.El{}
	if len(atts) > 0 {
		boxKids = append(boxKids, giode.Row(widgets.St{Gap: 8, Wrap: giode.WrapWrap, Padding: giode.Inset{Bottom: 4}}, atts...))
	}
	c.in.Placeholder(placeholder)
	boxKids = append(boxKids,
		c.in,
		widgets.Row(widgets.St{Width: -1, Gap: 6}, append(tools, widgets.Spacer(), action)...),
	)
	st := widgets.St{
		FlexDirection: giode.FlexDirectionColumn, Width: -1, Gap: 6,
		Background: theme.T.Surface, BorderRadius: 24, BorderWidth: 1, BorderColor: theme.T.BorderStrong,
		Padding: giode.Inset{Top: 10, Left: 12, Right: 10, Bottom: 10},
	}
	// A blurred shadow is drawn as one translucent layer per dp of blur,
	// all redrawn every frame; keep it small, and skip it in dark mode
	// where it barely shows.
	if !theme.T.Dark {
		st.BoxShadow = giode.BoxShadow{Y: 2, Blur: 6, Color: theme.T.Shadow}
	}
	box := giode.Box(st, boxKids...)
	hint := "Pocket Agent can make mistakes. Check important info."
	if a.loadingLLM != "" {
		hint = "Loading the on-device model…"
	}
	return giode.Box(widgets.St{Width: -1, Padding: giode.Inset{Left: 12, Right: 12, Bottom: 8}},
		widgets.Centered(780, widgets.Col(widgets.St{Width: -1, Gap: 6, Align: giode.AlignCenter},
			box,
			widgets.Text(hint, theme.SizeMicro, theme.T.Faint),
		)))
}
