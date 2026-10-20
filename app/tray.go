package main

import (
	"log"

	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/egoist/mygo"
)

// trayScale draws the 12 x 12 sprite of the tray icon at 24 x 24, which is one to one at 150% scaling.
const trayScale = 2

// newTray puts the icon in the notification area of Windows. A click brings the main window back;
// the right button opens Open, Settings… and Exit. Closing the window hides it, so this and a second
// launch are the ways back. The caller destroys it on quit. It must run after the app is ready.
func (s *shell) newTray() *mygo.Tray {
	items := trayItems(s.showMain, func() { s.OpenSettings(views.PaneGeneral) })
	open, settings := items[0], items[1]
	tray, err := mygo.NewTray(mygo.TrayOptions{
		Icon:    theme.TrayIcon(trayScale),
		ToolTip: locales.WindowTitle(),
		Menu:    mygo.NewMenu(items),
	})
	if err != nil {
		log.Printf("tray: %v", err)
		return nil
	}
	tray.OnClick(s.showMain)
	mygo.App.OnLocaleChanged(func(string) {
		open.SetLabel(locales.TrayOpen())
		settings.SetLabel(locales.MenuSettings())
		tray.SetToolTip(locales.WindowTitle())
	})
	return tray
}

// trayItems are the items of the tray menu: Open brings the main window back, then Settings…, then
// Exit, which mygo names for the system.
func trayItems(open, settings func()) []*mygo.MenuItem {
	return []*mygo.MenuItem{
		{ID: "open", Label: locales.TrayOpen(), Click: func(*mygo.MenuItem, *mygo.Window) { open() }},
		{ID: "settings", Label: locales.MenuSettings(), Click: func(*mygo.MenuItem, *mygo.Window) { settings() }},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	}
}
