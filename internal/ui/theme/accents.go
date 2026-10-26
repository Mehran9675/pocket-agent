package theme

// Accent is a selectable highlight color.
type Accent struct {
	ID, Name        string
	Dark, Light     string
	OnDark, OnLight string
}

// Accents lists the accent choices.
var Accents = []Accent{
	{ID: "indigo", Name: "Indigo", Dark: "#7c83ff", Light: "#4f46e5", OnDark: "#0b0c10", OnLight: "#ffffff"},
	{ID: "teal", Name: "Teal", Dark: "#2dd4bf", Light: "#0d9488", OnDark: "#04201c", OnLight: "#ffffff"},
	{ID: "amber", Name: "Amber", Dark: "#f5b544", Light: "#c2740c", OnDark: "#1f1503", OnLight: "#ffffff"},
	{ID: "rose", Name: "Rose", Dark: "#fb7185", Light: "#e11d48", OnDark: "#26070d", OnLight: "#ffffff"},
	{ID: "sky", Name: "Sky", Dark: "#5cb8ff", Light: "#0273d4", OnDark: "#03182b", OnLight: "#ffffff"},
	{ID: "mono", Name: "Graphite", Dark: "#e8eaee", Light: "#16181d", OnDark: "#0b0c10", OnLight: "#ffffff"},
}

func accentByID(id string) Accent {
	for _, a := range Accents {
		if a.ID == id {
			return a
		}
	}
	return Accents[0]
}
