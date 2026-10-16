// Command adhd-todo is the desktop app: one window with the Mine and Agents screens and a list per project,
// drawn with mygo native UI. It reads and writes the same SQLite database as the adhd-core MCP
// server, so what agents report shows up live.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/egoist/mygo"
)

const version = "0.2.0"

// connectCommand registers the MCP server with Claude Code.
const connectCommand = "claude mcp add --scope user adhd-todo -- ~/.local/bin/adhd-core mcp"

func main() {
	fake := flag.Bool("fake", false, "use in-memory demo data instead of the database (for screenshots)")
	empty := flag.Bool("empty", false, "start with no data at all, to see the empty states (implies -fake)")
	flag.Parse()

	be, err := openBackend(*fake, *empty)
	if err != nil {
		log.Fatalf("open data: %v", err)
	}
	home, _ := os.UserHomeDir()
	app := views.NewApp(be.store, nil, views.Config{
		ConnectCommand: connectCommand,
		DataFolder:     be.dataFolder,
		Home:           home,
		Version:        version,
		AgentSetup:     agentsetup.New(home),
	})
	app.Assistant, app.Models, app.GitHub, app.Seen, app.Dismissed = be.assistant, be.models, be.github, be.seen, be.dismissed
	app.Prefs = be.prefs
	s := &shell{app: app, main: views.NewMain(app), settings: views.NewSettings(app), github: be.github}
	app.Act = s

	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { s.quitting.Store(true) })
	// Hiding the main window must not end the app.
	mygo.App.OnWindowAllClosed(func() {})
	// A click on the Dock icon with no window showing brings the main window back.
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			s.showMain()
		}
	})
	mygo.App.OnNotificationClick(s.openFromNotification)

	stop := make(chan struct{})
	mygo.App.OnQuit(func() {
		close(stop)
		if be.close != nil {
			_ = be.close()
		}
	})
	mygo.App.WhenReady(func() {
		s.setMenu()
		s.createMain()
		be.store.Subscribe(s.storeChanged)
		be.store.SubscribeReminders(func(r data.Reminder) { go notifyReminder(r) })
		if be.github != nil {
			be.github.Subscribe(s.githubChanged)
		}
		be.start(stop)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
