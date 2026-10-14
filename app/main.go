// Command adhd-todo is the menu bar app: a quick panel under the tray icon and a main window with
// the board, drawn with mygo native UI. For now it runs against an in-memory fake store; wiring to
// the real core is a later phase.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/egoist/mygo"
)

const version = "0.2.0"

func main() {
	empty := flag.Bool("empty", false, "start with no data, to see the empty states")
	_ = flag.Bool("fake", true, "use the in-memory fake store (the only store for now)")
	flag.Parse()

	var fake *data.Fake
	if *empty {
		fake = data.NewEmptyFake(time.Now)
	} else {
		fake = data.NewFake(time.Now)
	}
	app := views.NewApp(fake, nil, views.Config{
		ConnectCommand: "claude mcp add --scope user adhd-todo -- ~/.local/bin/adhd-core mcp",
		DataFolder:     "~/Library/Application Support/adhd-todo",
		Version:        version,
	})
	s := &shell{app: app, main: views.NewMain(app)}
	app.Act = s

	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { s.quitting.Store(true) })
	// The panel closing must not end the app, and neither must hiding the main window.
	mygo.App.OnWindowAllClosed(func() {})
	// A click on the Dock icon with no window showing brings the main window back.
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			s.showMain()
		}
	})
	mygo.App.OnNotificationClick(s.openFromNotification)

	stop := make(chan struct{})
	mygo.App.OnQuit(func() { close(stop) })
	mygo.App.WhenReady(func() {
		s.createMain()
		if err := s.createTray(); err != nil {
			log.Printf("tray: %v", err)
		}
		s.updateTrayIcon()
		fake.Subscribe(s.storeChanged)
		fake.SubscribeReminders(func(r data.Reminder) { go notifyReminder(r) })
		go tick(fake, stop)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// tick asks the fake to send due reminders; the real store does this from its change watcher.
func tick(f *data.Fake, stop <-chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			f.CheckReminders()
		case <-stop:
			return
		}
	}
}
