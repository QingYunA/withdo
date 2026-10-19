package main

import (
	"slices"

	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
)

// setMenu installs the menu bar: mygo's standard one, with Settings… (⌘,) in the app menu, where
// macOS puts it.
func (s *shell) setMenu() {
	settings := &mygo.MenuItem{
		ID: "settings", Label: locales.MenuSettings(), Accelerator: "CmdOrCtrl+,",
		Click: func(*mygo.MenuItem, *mygo.Window) { s.OpenSettings(views.PaneGeneral) },
	}
	appMenu := []*mygo.MenuItem{
		{Role: mygo.RoleAbout},
		mygo.Separator(),
		settings,
		mygo.Separator(),
		{Role: mygo.RoleServices},
		mygo.Separator(),
		{Role: mygo.RoleHide},
		{Role: mygo.RoleHideOthers},
		{Role: mygo.RoleUnhide},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	}
	// Check for Updates… follows About, in a build that can update itself.
	var check *mygo.MenuItem
	if mygo.Updater.Enabled() {
		check = updater.MenuItem()
		appMenu = slices.Insert(appMenu, 1, check)
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu, Submenu: appMenu},
		{Role: mygo.RoleFileMenu},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	})
	mygo.App.SetMenu(menu)
	mygo.App.OnLocaleChanged(func(string) {
		settings.SetLabel(locales.MenuSettings())
		if check != nil {
			check.SetLabel(updater.MenuItem().Label)
		}
	})
}
