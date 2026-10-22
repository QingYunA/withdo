package main

import (
	"log"
	"runtime"

	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/egoist/mygo"
)

// trayScale draws the 12 x 12 sprite of the tray icon at 24 x 24, which is one to one at 150% scaling.
// macScale does it at 36 x 36 for the menu bar, which draws icons 18 points tall: one to one at 2x.
const (
	trayScale = 2
	macScale  = 3
)

// newTray puts the icon in the menu bar (macOS) or the notification area (Windows). A click opens the
// panel under it (panel.go). On Windows the right button opens Open, Settings… and Exit; the Mac has
// the Dock for those. Closing the main window hides it, so the panel, this menu and a second launch
// are the ways back. The caller destroys it on quit. It must run after the app is ready.
func (s *shell) newTray() *mygo.Tray {
	opts := mygo.TrayOptions{ToolTip: locales.WindowTitle()}
	var open, settings *mygo.MenuItem
	if runtime.GOOS == "darwin" {
		// A template image follows the menu bar's appearance, light or dark.
		opts.Icon, opts.IconIsTemplate = theme.TrayTemplateIcon(macScale), true
	} else {
		items := trayItems(s.showMain, func() { s.OpenSettings(views.PaneGeneral) })
		open, settings = items[0], items[1]
		opts.Icon, opts.Menu = theme.TrayIcon(trayScale), mygo.NewMenu(items)
	}
	tray, err := mygo.NewTray(opts)
	if err != nil {
		log.Printf("tray: %v", err)
		return nil
	}
	s.mu.Lock()
	s.tray = tray
	s.mu.Unlock()
	tray.OnClick(s.togglePanel)
	mygo.App.OnLocaleChanged(func(string) {
		if open != nil {
			open.SetLabel(locales.TrayOpen())
			settings.SetLabel(locales.MenuSettings())
		}
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
