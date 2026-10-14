package main

import (
	"log"

	"github.com/egoist/mygo"
)

func main() {
	core := &Core{}
	mygo.Bind(core)

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
		mygo.NewWindow(mygo.WindowOptions{Title: "ADHD Todo", Width: 1100, Height: 720, URL: "/#/main"})
	})
	mygo.App.OnQuit(func() {
		if core.sidecar != nil {
			core.sidecar.Stop()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
