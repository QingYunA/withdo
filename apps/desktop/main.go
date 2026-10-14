package main

import (
	"log"

	"github.com/egoist/mygo"
)

func main() {
	s := &shell{}
	core := &Core{}
	mygo.Bind(core)
	mygo.Bind(Shell{s})

	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { s.quitting.Store(true) })
	// The panel closing must not end the app, and neither must hiding the main window.
	mygo.App.OnWindowAllClosed(func() {})
	// A click on the Dock icon with no window showing brings the main window back.
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			s.showMain("")
		}
	})
	mygo.App.OnQuit(func() {
		if core.sidecar != nil {
			core.sidecar.Stop()
		}
	})

	mygo.App.WhenReady(func() {
		// The sidecar starts here, not at the top of main: `mygo generate` runs main up to Run.
		path, err := sidecarPath()
		if err != nil {
			log.Fatal(err)
		}
		core.sidecar = NewSidecar(SidecarConfig{
			Path:     path,
			Args:     []string{"serve"},
			Logf:     log.Printf,
			OnFailed: func(reason string) { _ = CoreError.Broadcast(reason) },
		})
		core.sidecar.Start()
		s.createMain()
		if _, err := s.createTray(); err != nil {
			log.Printf("tray: %v", err)
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
