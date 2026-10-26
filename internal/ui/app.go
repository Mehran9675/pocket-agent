// Package ui is Pocket Agent's interface, built with Giode: a responsive
// shell (drawer on phones, sidebar on tablets, sidebar and inspector on
// wide iPads), the chat, agent and research views, the model library and
// settings. Each view lives in its own file; shared components are in
// package widgets, design tokens in theme and Markdown in markdown.
package ui

import (
	"context"
	"sync"
	"time"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/components/drawer"
	"github.com/mehran9675/giode/components/input"
	"github.com/mehran9675/giode/components/scroll"
	"github.com/mehran9675/pocket-agent/internal/local"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/markdown"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// Version is the app version shown in About.
const Version = "1.0.0"

// App is the whole application state. Everything here is read and
// written on the UI goroutine; background work hands results back with
// Post.
type App struct {
	win *giode.App
	st  *store.Store

	settings *store.Settings
	convs    []*store.Conversation
	agents   []*store.Agent
	current  *store.Conversation
	draft    draft

	runs map[string]*runState

	engine     local.Engine
	models     *local.Store
	localProv  *local.Provider
	loadingLLM string
	ram        uint64

	mcp       map[string]*tools.MCPClient
	mcpTools  map[string][]*tools.Tool
	mcpStatus map[string]string

	// navigation: "" is the chat; other routes are pages.
	route   string
	history []string

	drawerOpen bool
	drawer     *drawer.Drawer
	sheet      *sheet
	confirm    *confirmState
	toasts     *giode.ToasterElement

	composer *composer
	chat     *chatView

	docs     *giode.KeyedCache[string, *markdown.Doc]
	expanded map[string]bool
	inputs   map[string]*input.Input
	scrolls  map[string]*scroll.Scroll
	sidebar  *sidebarView
	refresh  map[string]bool // providers whose model list is loading

	providerStatus map[string]string
	hfResults      []local.HFModel
	hfFiles        map[string][]local.HFFile
	hfBusy         bool
	hfErr          string
}

// draft holds the choices for a chat not started yet.
type draft struct {
	mode    store.Mode
	agentID string
	web     bool
	depth   store.ResearchDepth
	model   store.ModelRef
}

// runState tracks a conversation's running turn.
type runState struct {
	cancel    context.CancelFunc
	mu        sync.Mutex
	approvals map[string]chan bool
	plan      []tools.PlanItem
}

// New wires the app around a Giode window and the data directory.
func New(win *giode.App, st *store.Store) *App {
	a := &App{
		win:            win,
		st:             st,
		settings:       st.LoadSettings(),
		convs:          st.LoadConversations(),
		agents:         st.LoadAgents(),
		runs:           map[string]*runState{},
		engine:         local.NewEngine(),
		mcp:            map[string]*tools.MCPClient{},
		mcpTools:       map[string][]*tools.Tool{},
		mcpStatus:      map[string]string{},
		docs:           giode.NewKeyedCache[string, *markdown.Doc]().Retain(600),
		expanded:       map[string]bool{},
		inputs:         map[string]*input.Input{},
		scrolls:        map[string]*scroll.Scroll{},
		refresh:        map[string]bool{},
		providerStatus: map[string]string{},
		hfFiles:        map[string][]local.HFFile{},
		ram:            local.DeviceMemory(),
	}
	a.draft = draft{mode: store.ModeChat, depth: store.DepthStandard, model: a.settings.DefaultModel}
	a.models = local.NewStore(st.ModelsDir())
	a.models.OnChange = func() { win.Post(func() {}) }
	a.localProv = &local.Provider{
		Engine: a.engine,
		Store:  a.models,
		Options: func() (local.LoadOptions, local.GenParams) {
			ls := a.settings.Local
			g := local.DefaultGenParams()
			if ls.Temperature > 0 {
				g.Temperature = float32(ls.Temperature)
			}
			if ls.MaxTokens > 0 {
				g.MaxTokens = ls.MaxTokens
			}
			return local.LoadOptions{ContextSize: ls.ContextSize, Threads: ls.Threads, GPULayers: ls.GPULayers}, g
		},
		OnLoad: func(model string, loading bool) {
			win.Post(func() {
				if loading {
					a.loadingLLM = model
				} else {
					a.loadingLLM = ""
				}
			})
		},
	}
	a.toasts = giode.Toaster().Placement(giode.ToastTopCenter).Duration(3 * time.Second).Max(2)
	a.composer = newComposer(a)
	a.chat = newChatView(a)
	a.sidebar = newSidebar(a)
	a.drawer = giode.Drawer(&a.drawerOpen, giode.SideLeft, 300, func() giode.Element {
		return giode.Box(widgets.St{Width: -1, Height: -1, Background: theme.T.Sidebar}, a.sidebar.view(true))
	}, widgets.St{Background: "transparent", Padding: giode.UniformInset(0)})
	a.ensureDefaultModel()
	a.connectMCP()
	return a
}

// Theme re-applies the theme from settings.
func (a *App) Theme() {
	theme.ApplyTheme(a.settings.Theme, a.settings.Accent)
}

// Post runs fn on the UI goroutine.
func (a *App) Post(fn func()) { a.win.Post(fn) }

// SaveAll flushes everything to disk; call when going to background.
func (a *App) SaveAll() {
	done := make(chan struct{})
	go func() {
		a.st.Flush()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func (a *App) saveSettings() { a.st.SaveSettings(a.settings) }

func (a *App) toast(kind giode.ToastKind, msg string) { a.toasts.Show(kind, msg) }
