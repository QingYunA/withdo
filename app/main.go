// Command adhd-todo is the desktop app: one window with the Mine and Agents screens and a list per project,
// drawn with mygo native UI. It reads and writes the same SQLite database as the adhd-core MCP
// server, so what agents report shows up live. The same executable is adhd-core too (cli.go).
package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/app/views"
	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/QingYunA/adhd-todo/internal/cli"
	"github.com/QingYunA/adhd-todo/internal/speech"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/plugins/updater/native"
)

const version = "0.7.3"

// connectCommand registers the MCP server with Claude Code. core is the adhd-core path the agents
// run: shown under ~ outside Windows, where neither PowerShell nor cmd expands it, and there in
// quotes, since the install folder has a space.
func connectCommand(goos, core, home string) string {
	if goos == "windows" {
		core = `"` + core + `"`
	} else if rest, ok := strings.CutPrefix(core, home+"/"); ok {
		core = "~/" + rest
	}
	return "claude mcp add --scope user adhd-todo -- " + core + " mcp"
}

func main() {
	if args, ok := cliArgs(os.Args); ok {
		os.Exit(cli.Main(args, cli.Options{Secrets: newSyncSecrets()}))
	}
	fake := flag.Bool("fake", false, "use demo data in a temporary database instead of the real one (for screenshots)")
	empty := flag.Bool("empty", false, "start with no data at all, to see the empty states (implies -fake)")
	flag.Parse()

	// A second launch hands over to the running app and ends here, before it opens the database. The
	// demo modes use a database of their own and may run beside the real one.
	if !*fake && !*empty && !mygo.App.RequestSingleInstanceLock() {
		return
	}
	be, err := openBackend(*fake, *empty)
	if err != nil {
		log.Fatalf("open data: %v", err)
	}
	home, _ := os.UserHomeDir()
	setup := agentsetup.New(home)
	app := views.NewApp(be.store, nil, views.Config{
		ConnectCommand:  connectCommand(runtime.GOOS, setup.CorePath(), home),
		DataFolder:      be.dataFolder,
		AssistantMemory: be.memoryFile,
		Home:            home,
		Version:         version,
		AgentSetup:      setup,
	})
	app.Assistant, app.Models, app.GitHub, app.Seen, app.Dismissed = be.assistant, be.models, be.github, be.seen, be.dismissed
	app.Prefs = be.prefs
	app.Recent, app.RecentCache = be.recent, be.recentCache
	s := &shell{app: app, main: views.NewMain(app), settings: views.NewSettings(app), popover: views.NewPopover(app), github: be.github}
	app.Act = s
	app.Sync = be.sync
	app.Recorder = speech.New()
	if be.sync != nil {
		app.Transcriber = be.sync
		be.sync.SetSyncNotify(func() { s.Post(func() {}) })
	}
	app.Sound, app.SystemSounds, app.WindowKey = theme.LoadVoices(), uiSoundsOn, s.windowKey
	s.notifier = s.newNotifier()

	// The update window is native UI like the app's own; its choices live in the plugin's updater.json.
	// A build without updates in mygo.json (or a dev build) cannot update, so it offers nothing.
	if mygo.Updater.Enabled() {
		mygo.Use(native.Plugin)
		updater.OnChange(s.invalidateSettings)
	}
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
	// The window is only hidden when it is closed, and Windows has no Dock to click, so launching the
	// app again brings the window back (the tray does too, below).
	mygo.App.OnSecondInstance(func([]string, string) { s.showMain() })

	stop := make(chan struct{})
	// The icon in the menu bar (macOS) or the tray (Windows) opens the panel; so does Option+Space.
	var tray *mygo.Tray
	mygo.App.OnQuit(func() {
		close(stop)
		mygo.GlobalShortcut.UnregisterAll()
		if tray != nil {
			tray.Destroy()
		}
		_ = be.close()
	})
	mygo.App.WhenReady(func() {
		s.setMenu()
		go linkOwnExecutable(app.Config.AgentSetup.CorePath())
		s.createMain()
		if tray = s.newTray(); tray != nil {
			s.registerShortcut()
		}
		be.store.Subscribe(s.storeChanged)
		be.store.SubscribeReminders(func(r data.Reminder) { go s.notifier.Reminder(r) })
		be.store.SubscribeEvents(func(e data.Event) { go s.notifier.Event(e) })
		if be.github != nil {
			be.github.Subscribe(s.githubChanged)
		}
		be.start(stop)
		go s.runEndOfDay(s.newEndOfDay(), stop)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
