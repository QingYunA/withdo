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

	// quitting is set when a quit starts, so that closing the main window then really closes it.
	quitting atomic.Bool
}

var _ views.Actions = (*shell)(nil)

func (s *shell) OpenTask(id int) {
	s.showMain()
	s.main.OpenTask(id)
	s.invalidateMain()
}

func (s *shell) OpenSettings() {
	s.showMain()
	s.main.OpenSettings()
	s.invalidateMain()
}

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

func (s *shell) invalidateMain() {
	if w := s.window(); w != nil {
		w.Invalidate()
	}
}

// createMain opens the main window. Closing it hides it: the app keeps running in the Dock until
// Quit.
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
	if win.IsMinimized() {
		win.Restore()
	}
	win.Show()
	win.Focus()
	mygo.App.Focus()
	s.app.Refresh()
}

// storeChanged runs on any goroutine after the data changed: reload the snapshot on the main thread
// and redraw the window.
func (s *shell) storeChanged() {
	win := s.window()
	if win == nil {
		return
	}
	win.Update(s.app.Refresh)
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
