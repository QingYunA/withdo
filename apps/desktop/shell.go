package main

import (
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
)

// MainNavigate asks the main window's page to show a hash route such as #/main/task/5.
var MainNavigate = mygo.NewEvent[string]("main-navigate")

// shell owns the windows. Callbacks run on the main thread, bound methods on their own
// goroutines, so the pointers are guarded; mygo is never called with the lock held.
type shell struct {
	mu    sync.Mutex
	main  *mygo.Window
	panel *mygo.Window

	// quitting is set when a quit starts, so that closing the main window then really closes it.
	quitting atomic.Bool
}

// Shell is the page's handle on the windows.
type Shell struct{ s *shell }

// OpenMain shows the main window and navigates it to hash, such as "#/main/task/5".
func (b Shell) OpenMain(hash string) { b.s.showMain(hash) }

func (s *shell) mainWindow() *mygo.Window {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.main
}

func (s *shell) panelWindow() *mygo.Window {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.panel
}

// createMain opens the main window. Closing it hides it: the app keeps running in the Dock and
// the menu bar until Quit.
func (s *shell) createMain() {
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:     "ADHD Todo",
		Width:     1100,
		Height:    720,
		MinWidth:  800,
		MinHeight: 520,
		StateKey:  "main",
		URL:       "/#/main",
	})
	win.OnClose(func(e *mygo.CloseEvent) {
		if !s.quitting.Load() {
			e.PreventDefault()
			win.Hide()
		}
	})
	s.mu.Lock()
	s.main = win
	s.mu.Unlock()
}

// showMain brings the main window to the front, optionally on a route.
func (s *shell) showMain(hash string) {
	win := s.mainWindow()
	if win == nil {
		return
	}
	s.closePanel()
	if hash != "" {
		_ = MainNavigate.Emit(win, hash)
	}
	if win.IsMinimized() {
		win.Restore()
	}
	win.Show()
	win.Focus()
	mygo.App.Focus()
}
