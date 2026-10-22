package main

import (
	"log"

	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// panelShortcut is the global shortcut that opens the panel: Option+Space on a Mac, Alt+Space on
// Windows (mygo's Alt is the Option key there).
const panelShortcut = "Alt+Space"

// registerShortcut makes the shortcut toggle the panel. Another app may hold the keys: that is
// logged, and the icon still works.
func (s *shell) registerShortcut() {
	if err := mygo.GlobalShortcut.Register(panelShortcut, s.togglePanel); err != nil {
		log.Printf("global shortcut %s: %v", panelShortcut, err)
	}
}

// panelHeight is the height the panel has before its content has been drawn once; the content asks
// for the real one.
const panelHeight = 320

// togglePanel opens the panel, or closes it when it is open. It runs on the main thread.
func (s *shell) togglePanel() {
	s.mu.Lock()
	open := s.popWin != nil
	s.mu.Unlock()
	if open {
		s.closePanel()
		return
	}
	s.openPanel()
}

// openPanel shows the panel under the tray icon, with the composer focused. A flyout is gone once it
// closes, so each opening makes a new one over the same content, which keeps its draft.
func (s *shell) openPanel() {
	s.mu.Lock()
	tray := s.tray
	s.mu.Unlock()
	if tray == nil {
		return
	}
	s.popover.Close = s.closePanel
	s.popover.Resize = s.resizePanel
	s.popover.Show()
	height := s.popover.Height()
	if height == 0 {
		height = panelHeight
	}
	win := mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      tray,
		Placement: mygo.PlacementBottom,
		Gap:       4,
		Width:     int(views.PopoverWidth),
		Height:    height,
		Focusable: true,
		Shadow:    true,
		Popover:   true,
		Content:   ui.View(s.popover.View),
	})
	win.OnClosed(func() {
		s.mu.Lock()
		if s.popWin == win {
			s.popWin = nil
		}
		s.mu.Unlock()
	})
	s.mu.Lock()
	s.popWin = win
	s.mu.Unlock()
	// A panel opened from another app is not the active window of this one: take the keyboard.
	mygo.App.Focus()
}

// closePanel closes the panel. The content asks for it from inside its own frame, so the window is
// closed after that frame.
func (s *shell) closePanel() {
	s.mu.Lock()
	win := s.popWin
	s.mu.Unlock()
	if win != nil {
		win.Update(win.Close)
	}
}

// resizePanel makes the panel as tall as its content asks, after the frame that asked.
func (s *shell) resizePanel(height int) {
	s.mu.Lock()
	win := s.popWin
	s.mu.Unlock()
	if win != nil {
		win.Update(func() { win.SetContentSize(int(views.PopoverWidth), height) })
	}
}

// invalidatePanel redraws the panel when it is open: what it lists changed.
func (s *shell) invalidatePanel() {
	s.mu.Lock()
	win := s.popWin
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}
