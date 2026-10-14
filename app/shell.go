package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// shell owns the windows. Callbacks run on the main thread; mygo is never called with the lock
// held. It implements views.Actions.
type shell struct {
	app  *views.App
	main *views.Main

	mu      sync.Mutex
	mainWin *mygo.Window
	panel   *mygo.Window
	panelV  *views.Panel
	tray    *mygo.Tray
	blocked bool

	// quitting is set when a quit starts, so that closing the main window then really closes it.
	quitting atomic.Bool
}

var _ views.Actions = (*shell)(nil)

func (s *shell) OpenTask(id int) {
	s.showMain()
	s.main.OpenTask(id)
	s.invalidateMain()
}

func (s *shell) OpenMain() { s.showMain() }

func (s *shell) OpenSettings() {
	s.showMain()
	s.main.OpenSettings()
	s.invalidateMain()
}

func (s *shell) ClosePanel() { s.closePanel() }

func (s *shell) OpenURL(url string) {
	if err := mygo.Shell.OpenExternal(url); err != nil {
		log.Printf("open %s: %v", url, err)
	}
}

func (s *shell) window() *mygo.Window {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mainWin
}

func (s *shell) panelWindow() *mygo.Window {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.panel
}

func (s *shell) invalidateMain() {
	if w := s.window(); w != nil {
		w.Invalidate()
	}
}

// createMain opens the main window. Closing it hides it: the app keeps running in the Dock and the
// menu bar until Quit.
func (s *shell) createMain() {
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:          "ADHD Todo",
		Width:          theme.WindowWidth,
		Height:         theme.WindowHeight,
		MinWidth:       theme.WindowMinWidth,
		MinHeight:      theme.WindowMinHeight,
		StateKey:       "main",
		TitleBarStyle:  mygo.TitleBarHiddenInset,
		TitleBarHeight: 52,
		Vibrancy:       mygo.VibrancySidebar,
		Content:        ui.View(s.main.View),
	})
	win.OnClose(func(e *mygo.CloseEvent) {
		if !s.quitting.Load() {
			e.PreventDefault()
			win.Hide()
		}
	})
	s.mu.Lock()
	s.mainWin = win
	s.mu.Unlock()
}

// showMain brings the main window to the front.
func (s *shell) showMain() {
	win := s.window()
	if win == nil {
		return
	}
	s.closePanel()
	if win.IsMinimized() {
		win.Restore()
	}
	win.Show()
	win.Focus()
	mygo.App.Focus()
	s.app.Refresh()
}

// togglePanel opens the quick panel under the tray icon, or closes it when it is open. The panel is
// a focusable flyout: it closes on a click outside and on an Escape the view leaves alone.
func (s *shell) togglePanel() {
	if s.panelWindow() != nil {
		s.closePanel()
		return
	}
	s.app.Refresh()
	panel := views.NewPanel(s.app)
	win := mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      s.tray,
		Placement: mygo.PlacementBottom,
		Gap:       4,
		Width:     theme.PanelWidth,
		Height:    theme.PanelHeight,
		Focusable: true,
		Shadow:    true,
		Popover:   true, // macOS draws its material, which is Liquid Glass on macOS 26 and later
		Content:   ui.View(panel.View),
	})
	s.mu.Lock()
	s.panel, s.panelV = win, panel
	s.mu.Unlock()
	win.OnClosed(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.panel == win {
			s.panel, s.panelV = nil, nil
		}
	})
}

func (s *shell) closePanel() {
	if w := s.panelWindow(); w != nil {
		w.Close()
	}
}

// createTray puts the icon in the menu bar. A left click toggles the quick panel; a right click
// opens the menu. The menu is installed only for that popup: a menu set on the tray would take the
// left click as well.
func (s *shell) createTray() error {
	tray, err := mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(false), IconIsTemplate: true, ToolTip: "ADHD Todo"})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.tray = tray
	s.mu.Unlock()
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Open main window", Click: func(*mygo.MenuItem, *mygo.Window) { s.showMain() }},
		{Label: "Settings…", Click: func(*mygo.MenuItem, *mygo.Window) { s.OpenSettings() }},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	})
	tray.OnClick(s.togglePanel)
	tray.OnRightClick(func() {
		s.closePanel()
		tray.SetMenu(menu)
		defer tray.SetMenu(nil)
		tray.PopUpMenu()
	})
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { tray.Destroy() })
	return nil
}

// storeChanged runs on any goroutine after the data changed: reload the snapshot on the main thread,
// redraw the windows and update the tray glyph.
func (s *shell) storeChanged() {
	win := s.window()
	if win == nil {
		return
	}
	win.Update(func() {
		s.app.Refresh()
		if p := s.panelWindow(); p != nil {
			p.Invalidate()
		}
		s.updateTrayIcon()
	})
}

// updateTrayIcon shows the dot while an agent task is blocked.
func (s *shell) updateTrayIcon() {
	s.mu.Lock()
	tray, was := s.tray, s.blocked
	s.mu.Unlock()
	now := s.app.AgentBlocked()
	if tray == nil || now == was {
		return
	}
	if err := tray.SetIcon(trayIcon(now), true); err != nil {
		log.Printf("tray icon: %v", err)
		return
	}
	s.mu.Lock()
	s.blocked = now
	s.mu.Unlock()
}

// notifyReminder shows a system notification for a due reminder. The first one asks the user for
// permission and blocks until they answer, so it must not run on the main thread.
func notifyReminder(r data.Reminder) {
	if !mygo.NotificationsSupported() {
		log.Printf("notifications are not available (is the app packaged?)")
		return
	}
	body := r.ProjectName
	if body == "" {
		body = "Inbox"
	}
	n := mygo.NewNotification(mygo.NotificationOptions{ID: notificationID(r.TaskID), Title: "Reminder: " + r.Title, Body: body})
	if err := n.Show(); err != nil {
		log.Printf("notification: %v", err)
	}
}

func notificationID(taskID int) string { return fmt.Sprintf("task/%d", taskID) }

// taskFromNotification reverses notificationID.
func taskFromNotification(id string) (int, bool) {
	var n int
	if _, err := fmt.Sscanf(id, "task/%d", &n); err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// openFromNotification handles a click on a reminder, also one from an earlier run.
func (s *shell) openFromNotification(id string) {
	if task, ok := taskFromNotification(id); ok {
		s.OpenTask(task)
	}
}
