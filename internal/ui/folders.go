package ui

import (
	"strings"

	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

// moveToFolder puts c in the folder with id ("" for none).
func (a *App) moveToFolder(c *store.Conversation, id string) {
	c.Folder = id
	c.Version++
	a.st.SaveConversation(c)
	if f := a.settings.Folder(id); f != nil && f.Collapsed {
		f.Collapsed = false
		a.saveSettings()
	}
}

// deleteFolder removes a folder; its conversations go back to the list.
func (a *App) deleteFolder(f *store.Folder) {
	for _, c := range a.convs {
		if c.Folder == f.ID {
			a.moveToFolder(c, "")
		}
	}
	for i, x := range a.settings.Folders {
		if x == f {
			a.settings.Folders = append(a.settings.Folders[:i], a.settings.Folders[i+1:]...)
			break
		}
	}
	a.saveSettings()
}

// openFolderName asks for a folder's name: a new folder when f is nil,
// which done receives.
func (a *App) openFolderName(f *store.Folder, done func(*store.Folder)) {
	title, init, key := tr("New folder"), "", "folder/new"
	if f != nil {
		title, init, key = tr("Rename folder"), f.Name, "folder/rename/"+f.ID
	}
	in := a.input(key, tr("Folder name"), init)
	in.SetText(init)
	save := func() {
		name := strings.TrimSpace(in.Text())
		if name == "" {
			return
		}
		if f == nil {
			f = &store.Folder{ID: store.NewID(), Name: name}
			a.settings.Folders = append(a.settings.Folders, f)
		} else {
			f.Name = name
		}
		a.saveSettings()
		a.closeSheet()
		if done != nil {
			done(f)
		}
	}
	in.Submit(func(string) { save() })
	sh := a.openSheet("foldername", title, func() widgets.El {
		return widgets.Col(widgets.St{Gap: 14, Width: -1}, in)
	})
	sh.footer = func() widgets.El {
		return widgets.Row(widgets.St{Gap: 10, Width: -1}, widgets.Spacer(),
			widgets.Button("folder/cancel", tr("Cancel"), "", widgets.BtnSecondary, a.closeSheet),
			widgets.Button("folder/save", ifStr(f == nil, tr("Create"), tr("Save")), "", widgets.BtnPrimary, save))
	}
	in.Focus()
}

// openMoveToFolder lets the user pick c's folder.
func (a *App) openMoveToFolder(c *store.Conversation) {
	a.openSheet("movefolder", tr("Move to folder"), func() widgets.El {
		check := func(on bool) widgets.El {
			if on {
				return widgets.Icon("check", 18, theme.T.Primary)
			}
			return nil
		}
		rows := []widgets.El{
			widgets.ListRow("move/none", "message-square", theme.T.Muted, tr("No folder"), tr("Show it in the main list"), check(c.Folder == ""), func() {
				a.closeSheet()
				a.moveToFolder(c, "")
			}),
		}
		for _, f := range a.settings.Folders {
			rows = append(rows, widgets.ListRow("move/"+f.ID, "folder", theme.T.Primary, f.Name, "", check(c.Folder == f.ID), func() {
				a.closeSheet()
				a.moveToFolder(c, f.ID)
			}))
		}
		rows = append(rows, widgets.ListRow("move/new", "folder-plus", theme.T.Primary, tr("New folder…"), "", nil, func() {
			a.openFolderName(nil, func(f *store.Folder) { a.moveToFolder(c, f.ID) })
		}))
		return widgets.Group(rows...)
	})
}

// openFolderMenu offers a folder's actions.
func (a *App) openFolderMenu(f *store.Folder) {
	a.openSheet("foldermenu", f.Name, func() widgets.El {
		return widgets.Col(widgets.St{Gap: 12, Width: -1},
			widgets.Group(
				widgets.ListRow("fm/rename", "pencil", theme.T.Primary, tr("Rename"), "", nil, func() { a.openFolderName(f, nil) }),
			),
			widgets.Group(
				widgets.ListRow("fm/delete", "trash-2", theme.T.Danger, tr("Delete folder"), tr("Its conversations move back to the list"), nil, func() {
					a.closeSheet()
					a.ask(tr("Delete “{name}”?", "name", f.Name), tr("The folder is removed; its conversations stay, in the main list."), tr("Delete folder"), true, func() { a.deleteFolder(f) })
				}),
			),
		)
	})
}

// folderName names c's folder for menus.
func folderName(a *App, c *store.Conversation) string {
	if f := a.settings.Folder(c.Folder); f != nil {
		return tr("In {folder}", "folder", f.Name)
	}
	return ""
}
