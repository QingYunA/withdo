package main

import (
	"log"

	"github.com/egoist/mygo"
)

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "ADHD Todo", Width: 1100, Height: 720, URL: "/#/main"})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
