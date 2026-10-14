package main

import "github.com/egoist/mygo"

const (
	panelWidth  = 440
	panelHeight = 520
)

// togglePanel opens the quick panel under the tray icon, or closes it when it is open. The panel
// is a focusable flyout: it closes on a click outside and on an Escape the page leaves alone.
func (s *shell) togglePanel(tray *mygo.Tray) {
	if s.panelWindow() != nil {
		s.closePanel()
		return
	}
	panel := mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      tray,
		Placement: mygo.PlacementBottom,
		Width:     panelWidth,
		Height:    panelHeight,
		Focusable: true,
		Shadow:    true,
		URL:       "/#/panel",
	})
	s.mu.Lock()
	s.panel = panel
	s.mu.Unlock()
	panel.OnClosed(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.panel == panel {
			s.panel = nil
		}
	})
}

func (s *shell) closePanel() {
	if panel := s.panelWindow(); panel != nil {
		panel.Close()
	}
}
