package main

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
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
	// settings is the content of the Settings window, kept between openings so that it comes back
	// on the pane it was on.
	settings *views.Settings
	// github is the pull request poller; nil when there is none.
	github data.GitHub

	mu      sync.Mutex
	mainWin *mygo.Window
	// settingsWin is the Settings window while it is open: there is one.
	settingsWin *mygo.Window
	// choice is the language picked in Settings during this run; chosen tells that one was.
	choice string
	chosen bool

	// quitting is set when a quit starts, so that closing the main window then really closes it.
	quitting atomic.Bool
}

var _ views.Actions = (*shell)(nil)

func (s *shell) OpenTask(id int) {
	s.showMain()
	s.main.OpenTask(id)
	s.invalidateMain()
}

// OpenSettings implements views.Actions: it shows the Settings window on a pane, opening it when
// it is not, and bringing the one that is to the front.
func (s *shell) OpenSettings(p views.Pane) {
	s.settings.Show(p)
	s.mu.Lock()
	win := s.settingsWin
	s.mu.Unlock()
	if win == nil {
		s.createSettings()
		return
	}
	if win.IsMinimized() {
		win.Restore()
	}
	win.Show()
	win.Focus()
	win.Invalidate()
}

// ShowInFinder implements views.Actions.
func (s *shell) ShowInFinder(path string) {
	if err := mygo.Shell.OpenPath(path); err != nil {
		log.Printf("show %s: %v", path, err)
	}
}

// openRunner runs the macOS open command with args. Tests replace it: they never launch an app.
var openRunner = func(args ...string) error {
	out, err := exec.Command("open", args...).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return errors.New(msg)
		}
	}
	return err
}

// folderApps are the agent apps that open a folder when given one; the others only come forward.
var folderApps = map[string]bool{"Cursor": true, "Windsurf": true, "Zed": true, "Visual Studio Code": true}

// editors are the editors an editor choice opens, the first one installed.
var editors = []string{"Cursor", "Visual Studio Code", "Zed"}

// OpenIn implements views.Actions. The agent's app is only brought forward: MCP gives no session
// id, so the exact session cannot be opened. An agent without an app, or one that is not
// installed, falls back to a terminal.
func (s *shell) OpenIn(how views.OpenWith, agentApp, dir string) error {
	switch how {
	case views.OpenTerminal:
		return openTerminal(dir)
	case views.OpenEditor:
		return openEditor(dir)
	case views.OpenFinder:
		if dir == "" {
			return views.ErrNoDir
		}
		return openRunner(dir)
	}
	if agentApp != "" {
		args := []string{"-a", agentApp}
		if folderApps[agentApp] && dir != "" {
			args = append(args, dir)
		}
		if err := openRunner(args...); err == nil {
			return nil
		}
	}
	return openTerminal(dir)
}

// openTerminal opens Terminal at dir.
func openTerminal(dir string) error {
	if dir == "" {
		return views.ErrNoDir
	}
	return openRunner("-a", "Terminal", dir)
}

// openEditor opens the first installed editor on dir; open -Ra finds an app without launching it.
func openEditor(dir string) error {
	if dir == "" {
		return views.ErrNoDir
	}
	for _, app := range editors {
		if openRunner("-Ra", app) == nil {
			return openRunner("-a", app, dir)
		}
	}
	return views.ErrNoEditor
}

func (s *shell) OpenURL(url string) {
	if err := mygo.Shell.OpenExternal(url); err != nil {
		log.Printf("open %s: %v", url, err)
	}
}

// Languages implements views.Actions: the languages the catalogs have.
func (s *shell) Languages() []string { return mygo.App.Languages() }

// Language implements views.Actions. mygo keeps the language the user chose but does not report
// it, so after a launch it is read back: a language that differs from the one the system's
// languages would give was chosen, and the same one is taken as following the system.
func (s *shell) Language() string {
	if s.chosen {
		return s.choice
	}
	shown := mygo.App.Locale()
	if shown == systemLanguage() {
		return ""
	}
	return shown
}

// SetLanguage implements views.Actions.
func (s *shell) SetLanguage(tag string) {
	s.choice, s.chosen = tag, true
	mygo.App.SetLocale(tag)
}

// systemLanguage is the app language the system's preferred languages give: the first one that
// names a language of the app, else the app's own.
func systemLanguage() string {
	langs := mygo.App.Languages()
	for _, preferred := range mygo.App.PreferredLanguages() {
		for _, l := range langs {
			if preferred == l || strings.HasPrefix(preferred, l+"-") {
				return l
			}
		}
	}
	return langs[0]
}

// Post runs fn on the main thread and redraws; the assistant and the connection test answer
// through it, so the Settings window redraws too.
func (s *shell) Post(fn func()) {
	if w := s.window(); w != nil {
		w.Update(func() {
			fn()
			s.invalidateSettings()
		})
	}
}

func (s *shell) invalidateSettings() {
	s.mu.Lock()
	win := s.settingsWin
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
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
		Title:          locales.WindowTitle(),
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
	// What was set once says the new language when it changes.
	mygo.App.OnLocaleChanged(func(string) { win.SetTitle(locales.WindowTitle()) })
	// Coming back to the front checks the pull requests again when the last check is old.
	win.OnFocus(func() {
		if s.github != nil {
			s.github.Activate()
		}
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

// createSettings opens the Settings window: one fixed size for every pane (a pane that needs more
// room scrolls), and the title and the pane icons in one band under the traffic lights, drawn by
// the view over a hidden title bar. The window's own title is still the open pane's name. Closing
// it destroys it.
func (s *shell) createSettings() {
	var win *mygo.Window
	s.settings.OnTitle = func(title string) {
		if win != nil {
			win.SetTitle(title)
		}
	}
	w, h := s.settings.ContentSize()
	win = mygo.NewWindow(mygo.WindowOptions{
		Title:           locales.SettingsGeneral(),
		Width:           int(w),
		Height:          int(h),
		UseContentSize:  true,
		DisableResize:   true,
		DisableMaximize: true,
		TitleBarStyle:   mygo.TitleBarHiddenInset,
		TitleBarHeight:  int(views.SettingsTitleBarHeight),
		Content:         ui.View(s.settings.View),
	})
	win.OnClosed(func() {
		s.mu.Lock()
		s.settingsWin = nil
		s.mu.Unlock()
	})
	s.mu.Lock()
	s.settingsWin = win
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
	win.Update(func() {
		s.app.Refresh()
		s.invalidateSettings()
	})
}

// githubChanged runs on any goroutine when the poller's status changed: redraw, so that Settings
// shows it. The tasks it changed arrive through storeChanged.
func (s *shell) githubChanged() { s.Post(func() {}) }

// notifyReminder shows a system notification for a due reminder. The first one asks the user for
// permission and blocks until they answer, so it must not run on the main thread.
func notifyReminder(r data.Reminder) {
	if !mygo.NotificationsSupported() {
		log.Printf("notifications are not available (is the app packaged?)")
		return
	}
	body := r.ProjectName
	if body == "" {
		body = locales.NavInbox()
	}
	n := mygo.NewNotification(mygo.NotificationOptions{ID: notificationID(r.TaskID), Title: locales.NotifyReminder(r.Title), Body: body})
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
