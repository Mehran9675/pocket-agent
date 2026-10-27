// Pocket Agent is a private AI workspace for iPhone, iPad and Android:
// chat with on-device or cloud models, run deep research and hand
// multi-step tasks to agents. The interface is built with Giode.
package main

import (
	"image"
	"log"
	"os"
	"path/filepath"

	"gioui.org/app"

	"github.com/mehran9675/giode"

	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
)

func main() {
	st, err := openStore()
	if err != nil {
		log.Fatal(err)
	}
	if err := theme.RegisterFonts(); err != nil {
		log.Fatal(err)
	}
	if err := theme.RegisterIcons(); err != nil {
		log.Fatal(err)
	}
	settings := st.LoadSettings()
	theme.ApplyTheme(settings.Theme, settings.Accent)

	var pa *ui.App
	win := giode.New(giode.Config{
		Title:      "Pocket Agent",
		Size:       image.Pt(1180, 820),
		MinSize:    image.Pt(360, 560),
		Resizable:  true,
		Background: "theme.background",
		OnBackground: func() {
			if pa != nil {
				pa.SaveAll()
			}
		},
		OnShutdown: func() {
			if pa != nil {
				pa.SaveAll()
			}
		},
	})
	pa = ui.New(win, st)
	if err := win.Run(pa.View); err != nil {
		log.Fatal(err)
	}
}

// openStore opens the data directory, falling back to the system temp
// directory so the app still starts if the usual location is unusable.
func openStore() (*store.Store, error) {
	dir, err := dataDir()
	if err == nil {
		var st *store.Store
		if st, err = store.Open(dir); err == nil {
			return st, nil
		}
	}
	log.Printf("pocket-agent: data directory unavailable (%v); using a temporary one", err)
	return store.Open(filepath.Join(os.TempDir(), "pocket-agent"))
}

// dataDir is the app's private storage: the app sandbox on iOS and
// Android, the user config directory on desktops. POCKET_AGENT_DATA
// overrides it.
func dataDir() (string, error) {
	if d := os.Getenv("POCKET_AGENT_DATA"); d != "" {
		return d, os.MkdirAll(d, 0o755)
	}
	base, err := app.DataDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(base, "pocket-agent")
	return d, os.MkdirAll(d, 0o755)
}
