package ui

import (
	"strings"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/giode/i18n"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// language is a UI language, named in itself.
type language struct{ code, name string }

// languages are the translations the app ships, by native name.
var languages = []language{
	{"en", "English"}, {"bg", "Български"}, {"cs", "Čeština"}, {"da", "Dansk"}, {"de", "Deutsch"},
	{"el", "Ελληνικά"}, {"es", "Español"}, {"et", "Eesti"}, {"fi", "Suomi"}, {"fr", "Français"},
	{"hr", "Hrvatski"}, {"hu", "Magyar"}, {"it", "Italiano"}, {"lt", "Lietuvių"}, {"lv", "Latviešu"},
	{"nb", "Norsk bokmål"}, {"nl", "Nederlands"}, {"pl", "Polski"}, {"pt", "Português"}, {"ro", "Română"},
	{"ru", "Русский"}, {"sk", "Slovenčina"}, {"sl", "Slovenščina"}, {"sv", "Svenska"}, {"uk", "Українська"},
}

// ApplyLanguage sets the UI locale: code, or the system's when empty.
func ApplyLanguage(code string) {
	if code == "" {
		code = i18n.SystemLocale()
		// Norwegian systems may say "no"; the translation is Bokmål.
		if code == "no" || strings.HasPrefix(code, "no-") {
			code = "nb"
		}
	}
	giode.SetLocale(code)
}

// languageName is the setting's label.
func languageName(code string) string {
	for _, l := range languages {
		if l.code == code {
			return l.name
		}
	}
	return tr("System")
}

func (a *App) setLanguage(code string) {
	a.settings.Language = code
	a.saveSettings()
	ApplyLanguage(code)
	giode.InvalidateCaches()
}

func (a *App) openLanguageSheet() {
	row := func(code, name, sub string) widgets.El {
		var check widgets.El
		if a.settings.Language == code {
			check = widgets.Icon("check", 18, theme.T.Primary)
		}
		return widgets.ListRow("lang/"+ifStr(code == "", "system", code), "globe", theme.T.Primary, name, sub, check, func() {
			a.closeSheet()
			a.setLanguage(code)
		})
	}
	a.openSheet("language", tr("Language"), func() widgets.El {
		rows := []widgets.El{row("", tr("System"), tr("Follow the device's language"))}
		for _, l := range languages {
			rows = append(rows, row(l.code, l.name, ""))
		}
		return widgets.Group(rows...)
	})
}
