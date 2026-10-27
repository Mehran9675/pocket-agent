package ui

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

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
	// full shows the message being written full screen.
	full bool
}

func newComposer(a *App) *composer {
	c := &composer{a: a, thumbs: map[string]image.Image{}}
	// Grows from one to eight lines. With a hardware keyboard Enter
	// sends and Shift+Enter starts a new line; phones send with the
	// button.
	c.in = giode.Input.Text("Message Pocket Agent", inputStyles(false)).
		Lines(1, 8).SubmitOn(giode.SubmitEnterUnlessShift).Label("Message")
	c.in.Submit(func(string) { c.submit() })
	return c
}

// inputStyles styles the message field, in the composer or full screen.
func inputStyles(full bool) widgets.St {
	st := widgets.St{
		Background: "transparent", BorderColor: "transparent", Padding: giode.Inset{Top: 6, Bottom: 6, Left: 4, Right: 4},
		Color: "theme.text", FontSize: theme.SizeBody,
	}
	if full {
		st.Height = -1
	}
	return st
}

// setFull switches between the composer and the full-screen editor,
// keeping the text.
func (c *composer) setFull(on bool) {
	c.full = on
	c.in.Styles(inputStyles(on))
	if on {
		c.in.Lines(1, 0)
		c.in.Focus()
	} else {
		c.in.Lines(1, 8)
	}
}

// fullView is the full-screen editor for long messages.
func (c *composer) fullView() widgets.El {
	a := c.a
	giode.HandleBack(func() { c.setFull(false) })
	canSend := !c.busy() && (c.in.Text() != "" || len(c.pending) > 0)
	send := widgets.Button("composer/full/send", "Send", "arrow-up", widgets.BtnPrimary, func() {
		if canSend {
			c.setFull(false)
			c.submit()
		}
	})
	if !canSend || a.isRunning(a.current) {
		send = widgets.Button("composer/full/send", "Send", "arrow-up", widgets.BtnSecondary, nil)
	}
	words := len(strings.Fields(c.in.Text()))
	return widgets.Col(widgets.St{Width: -1, Height: -1, Gap: 8, Padding: giode.Inset{Left: 12, Right: 12, Top: 8, Bottom: 12}},
		widgets.Row(widgets.St{Width: -1, Gap: 8},
			widgets.IconButton("composer/collapse", "minimize-2", "Collapse", 20, theme.T.Text2, func() { c.setFull(false) }),
			giode.Box(widgets.St{FlexGrow: 1}, widgets.Text(fmt.Sprintf("%d words", words), theme.SizeFootnote, theme.T.Muted)),
			widgets.IconButton("composer/full/attach", "plus", "Attach", 20, theme.T.Text2, a.openAttachSheet),
			send),
		giode.Box(widgets.St{
			Width: -1, FlexGrow: 1, FlexDirection: giode.FlexDirectionColumn, Padding: giode.UniformInset(10),
			Background: theme.T.Surface, BorderRadius: 18, BorderWidth: 1, BorderColor: theme.T.BorderStrong,
		}, c.in))
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
		widgets.Row(widgets.St{Width: -1, Gap: 6}, append(tools, widgets.Spacer(),
			widgets.IconButton("composer/expand", "maximize-2", "Full screen", 18, theme.T.Text2, func() { c.setFull(true) }),
			action)...),
	)
	st := widgets.St{
		FlexDirection: giode.FlexDirectionColumn, Width: -1, Gap: 6,
		Background: theme.T.Surface, BorderRadius: 24, BorderWidth: 1, BorderColor: theme.T.BorderStrong,
		Padding: giode.Inset{Top: 10, Left: 12, Right: 10, Bottom: 10},
	}
	// Giode draws blurred shadows from one cached nine-slice image, so a
	// soft shadow costs little.
	st.BoxShadow = giode.BoxShadow{Y: 6, Blur: 24, Color: theme.T.Shadow}
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
