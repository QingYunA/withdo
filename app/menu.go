package main

import (
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/egoist/mygo"
)

// setMenu installs the menu bar: mygo's standard one, with Settings… (⌘,) in the app menu, where
// macOS puts it.
func (s *shell) setMenu() {
	settings := &mygo.MenuItem{
		ID: "settings", Label: locales.MenuSettings(), Accelerator: "CmdOrCtrl+,",
		Click: func(*mygo.MenuItem, *mygo.Window) { s.OpenSettings(views.PaneGeneral) },
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu, Submenu: []*mygo.MenuItem{
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
		}},
		{Role: mygo.RoleFileMenu},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	})
	mygo.App.SetMenu(menu)
	mygo.App.OnLocaleChanged(func(string) { settings.SetLabel(locales.MenuSettings()) })
}
