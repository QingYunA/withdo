package main

import (
	"runtime"
	"slices"

	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
)

// setMenu installs the menu bar: mygo's standard one, with Settings… (⌘,) where the system puts it.
func (s *shell) setMenu() {
	settings := &mygo.MenuItem{
		ID: "settings", Label: locales.MenuSettings(), Accelerator: "CmdOrCtrl+,",
		Click: func(*mygo.MenuItem, *mygo.Window) { s.OpenSettings(views.PaneGeneral) },
	}
	// Check for Updates… is in a build that can update itself.
	var check *mygo.MenuItem
	if mygo.Updater.Enabled() {
		check = updater.MenuItem()
	}
	mygo.App.SetMenu(mygo.NewMenu(menuItems(runtime.GOOS, settings, check)))
	mygo.App.OnLocaleChanged(func(string) {
		settings.SetLabel(locales.MenuSettings())
		if check != nil {
			check.SetLabel(updater.MenuItem().Label)
		}
	})
}

// menuItems are the menus of the menu bar. On a Mac Settings… is in the app menu, where macOS puts
// it, and Check for Updates… (check, nil in a build that cannot update) follows About. Elsewhere
// mygo shows no app menu, so both go in the File menu above Exit, as VS Code does.
func menuItems(goos string, settings, check *mygo.MenuItem) []*mygo.MenuItem {
	if goos == "darwin" {
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
		if check != nil {
			appMenu = slices.Insert(appMenu, 1, check)
		}
		return []*mygo.MenuItem{
			{Role: mygo.RoleAppMenu, Submenu: appMenu},
			{Role: mygo.RoleFileMenu},
			{Role: mygo.RoleEditMenu},
			{Role: mygo.RoleViewMenu},
			{Role: mygo.RoleWindowMenu},
		}
	}
	file := []*mygo.MenuItem{settings}
	if check != nil {
		file = append(file, check)
	}
	file = append(file, mygo.Separator(), &mygo.MenuItem{Role: mygo.RoleQuit})
	return []*mygo.MenuItem{
		{Role: mygo.RoleFileMenu, Submenu: file},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	}
}
