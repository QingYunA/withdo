package main

import (
	_ "embed"

	"github.com/egoist/mygo"
)

//go:embed assets/tray.png
var trayIcon []byte

// createTray puts the icon in the menu bar. A left click toggles the quick panel; a right click
// opens the menu. The menu is installed only for that popup: a menu set on the tray would take
// the left click as well.
func (s *shell) createTray() (*mygo.Tray, error) {
	tray, err := mygo.NewTray(mygo.TrayOptions{Icon: trayIcon, IconIsTemplate: true, ToolTip: "ADHD Todo"})
	if err != nil {
		return nil, err
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Open main window", Click: func(*mygo.MenuItem, *mygo.Window) { s.showMain("#/main") }},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	})
	tray.OnClick(func() { s.togglePanel(tray) })
	tray.OnRightClick(func() {
		s.closePanel()
		tray.SetMenu(menu)
		defer tray.SetMenu(nil)
		tray.PopUpMenu()
	})
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { tray.Destroy() })
	return tray, nil
}
