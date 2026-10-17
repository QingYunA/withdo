package views

import (
	"image"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// paintedIn reports whether a pixel within tol (per channel) of want is drawn in r, a rectangle
// in points of a window that is width points wide.
func paintedIn(img *image.RGBA, width float32, r ui.Rect, want ui.Color, tol int) bool {
	scale := float32(img.Bounds().Dx()) / width
	near := func(a, b uint8) bool { return absInt(int(a)-int(b)) <= tol }
	for y := int(r.Y * scale); y < int((r.Y+r.H)*scale); y++ {
		for x := int(r.X * scale); x < int((r.X+r.W)*scale); x++ {
			if !(image.Point{X: x, Y: y}).In(img.Bounds()) {
				continue
			}
			p := img.RGBAAt(x, y)
			if near(p.R, want.R) && near(p.G, want.G) && near(p.B, want.B) {
				return true
			}
		}
	}
	return false
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// paintedLeftOf reports whether the color is drawn within w points left of the text, on its lines.
func paintedLeftOf(t *testing.T, tt *ui.Tester, width float32, text string, w float32, want ui.Color) bool {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no %q in %q", text, tt.Texts())
	}
	return paintedIn(tt.Image(), width, ui.Rect{X: r.X - w, Y: r.Y - 8, W: w, H: r.H + 16}, want, 6)
}

var (
	youColor    = theme.You.Sprite().Palette['b']                        // the shirt
	claudeColor = theme.AgentAvatar("claude-code").Sprite().Palette['o'] // the body
	helperColor = theme.Assistant.Sprite().Palette['p']                  // the body of the assistant
)

func TestTaskWhoIsYouForMyTasksAndTheAgentForItsOwn(t *testing.T) {
	app, _, _ := newTestApp(t)
	mine, _ := app.Task(5)
	if mine.Source != data.SourceUser || app.taskWho(mine).Kind != theme.AvatarYou {
		t.Errorf("task 5 is mine, who %v", app.taskWho(mine).Kind)
	}
	agents, _ := app.Task(1)
	if agents.Source != data.SourceAgent || app.taskWho(agents).Kind != theme.AvatarClaude {
		t.Errorf("task 1 came from Claude Code, who %v (agent %q)", app.taskWho(agents).Kind, app.taskAgent(agents))
	}
	silent := data.Task{ID: 9999, Source: data.SourceAgent}
	if who := app.taskWho(silent); who.Kind != theme.AvatarOther || app.taskAgent(silent) != "agent" {
		t.Errorf("an agent that never reported is an unnamed robot, got %v %q", who.Kind, app.taskAgent(silent))
	}
}

func TestAgentNamesAreTheProductsNames(t *testing.T) {
	for in, want := range map[string]string{
		"claude-code": "Claude Code", "codex": "Codex", "Cursor": "Cursor", "gemini-cli": "Gemini CLI",
		"github": "GitHub", "agent": "Agent", "": "Agent", "opencode": "opencode",
		"Visual Studio Code": "Copilot (VS Code)", "qwen-code-mcp-client": "Qwen Code", "qwen-cli-mcp-client-x": "Qwen Code",
		"kimi-cli": "Kimi CLI", "Zed": "Zed", "windsurf": "Windsurf", "cline": "Cline", "optimized-agent": "optimized-agent",
		"deepseek-harness": "DeepSeek Harness", "dsh": "DeepSeek Harness", "dsh-desktop": "DeepSeek Harness",
		"amp": "Amp", "kiro": "Kiro",
	} {
		if got := agentName(in); got != want {
			t.Errorf("agentName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAgentRowsStartWithTheirAvatarAndMineHaveNone(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	if paintedLeftOf(t, tt, 1100, "Write the release notes", 30, youColor) {
		t.Error("my task has an avatar before its title: its column says whose it is")
	}
	if !paintedLeftOf(t, tt, 1100, "Port core to Go", 30, claudeColor) {
		t.Error("an agent's task has no avatar of Claude Code before its title")
	}
	// The rows sit in under the group's name: the title is a step further in than the name.
	row, _ := tt.Find("Write the release notes")
	group, _ := tt.Find("To do")
	if absInt(int(row.X-group.X-listIndent)) > 1 {
		t.Errorf("title at %.1f, group name at %.1f, want the title %.0f pt further in", row.X, group.X, listIndent)
	}
}

func TestBoardCardsStartTheirMetaLineWithTheAvatar(t *testing.T) {
	app, _, _ := newTestApp(t)
	tt, _ := boardTester(t, app, 1, 1100)
	card, _ := tt.Find("Write the release notes")
	// The meta line is under the title; the avatar is 12 pt at its left edge.
	region := ui.Rect{X: card.X, Y: card.Y + card.H, W: 40, H: 40}
	if !paintedIn(tt.Image(), 1100, region, youColor, 6) {
		t.Error("the card of my task has no avatar of mine in its meta line")
	}
}

func TestTheInspectorSaysWhoWroteTheTask(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.OpenTask(3) // an agent's
	tt.Frame()
	has(t, tt, "From", "Claude Code")
	from, _ := tt.Find("From")
	if !paintedIn(tt.Image(), 1100, ui.Rect{X: from.X + labelWidth - 28, Y: from.Y - 8, W: 30, H: from.H + 16}, claudeColor, 6) {
		t.Error("no Claude avatar before the name in the From row")
	}
	m.OpenTask(5) // mine
	tt.Frame()
	has(t, tt, "From", "You")
}

func TestEachReportOfTheActivityStartsWithItsAgentsAvatar(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.OpenTask(3)
	tt.Frame()
	if !paintedLeftOf(t, tt, 1100, "Reproduced the race with a slow launch", 40, claudeColor) {
		t.Error("the report has no avatar of its agent")
	}
}

func TestTheComposerStartsWithTheAssistantsAvatar(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	if !paintedLeftOf(t, tt, 1100, "Add a task, or ask me anything", 40, helperColor) {
		t.Error("the composer has no avatar of the assistant")
	}
}

func TestAChangeFreeAnswerStartsWithTheAssistantsAvatar(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Text: "That one is already in adhd-todo, so I didn't add it again."}, nil)})
	r.ask("write the release notes")
	r.answer(t)
	if !paintedLeftOf(t, r.tt, 1100, "That one is already in adhd-todo, so I didn't add it again.", 24, helperColor) {
		t.Error("the sentence does not start with the assistant's avatar")
	}
}

func TestTheSidebarHasAnIconForEveryRowAndTintsTheProjects(t *testing.T) {
	app, _, _ := newTestApp(t)
	m := NewMain(app)
	tt := ui.NewTester(m.View, 1100, 720)
	tt.SetScale(2)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	still(tt)
	tt.Frame()
	for _, p := range app.snap.projects {
		if !paintedLeftOf(t, tt, 1100, p.Name, 40, theme.ProjectColor(p.Name)) {
			t.Errorf("the folder of %q is not tinted its own color", p.Name)
		}
	}
}

func TestEmptyScreensShowYouAndTheAssistant(t *testing.T) {
	type screen struct {
		name string
		tt   func() *ui.Tester
		text string
	}
	empty, _ := newEmptyApp(t)
	projects, p := emptyProjectApp(t)
	screens := []screen{
		{"Now", func() *ui.Tester { tt, _, _ := newTesterFor(empty, 1100); return tt }, "Nothing needs you right now"},
		{"project", func() *ui.Tester {
			tt, m, _ := newTesterFor(projects, 1100)
			m.goTo(nav{screen: screenProject, projectID: p.ID})
			tt.Frame()
			return tt
		}, "No tasks yet."},
		{"Inbox", func() *ui.Tester {
			tt, m, _ := newTesterFor(empty, 1100)
			m.goTo(nav{screen: screenInbox})
			tt.Frame()
			return tt
		}, "Your inbox is clear."},
		{"Completed", func() *ui.Tester {
			tt, m, _ := newTesterFor(empty, 1100)
			m.goTo(nav{screen: screenCompleted})
			tt.Frame()
			return tt
		}, "Nothing finished yet"},
		{"search", func() *ui.Tester {
			tt, m, _ := newTesterFor(empty, 1100)
			m.manage.search.query = "zzzz"
			tt.Frame()
			return tt
		}, "No tasks match “zzzz”"},
	}
	for _, s := range screens {
		tt := s.tt()
		// Both avatars at 60 pt sit above the title.
		r, ok := tt.Find(s.text)
		if !ok {
			t.Errorf("%s: no %q in %q", s.name, s.text, tt.Texts())
			continue
		}
		above := ui.Rect{X: r.X - 100, Y: r.Y - 100, W: r.W + 200, H: 100}
		if !paintedIn(tt.Image(), 1100, above, youColor, 6) || !paintedIn(tt.Image(), 1100, above, helperColor, 6) {
			t.Errorf("%s: you and the assistant are not drawn above %q", s.name, s.text)
		}
	}
}

func TestTheDrawnMenuHasAnIconBeforeEveryItemAndDeleteIsRed(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.Frame()
	menuAt(tt, "Write the release notes", 120)
	for _, label := range []string{m.say.MenuMoveTo(), m.say.MenuRemindMe(), m.say.MenuAsk(), m.say.MenuDelete()} {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("no %q in %q", label, tt.Texts())
		}
		icon := ui.Rect{X: r.X, Y: r.Y, W: 34, H: r.H}
		if !paintedAnything(tt.Image(), 1100, icon) {
			t.Errorf("no icon before %q", label)
		}
	}
	del, _ := tt.Find(m.say.MenuDelete())
	if !paintedIn(tt.Image(), 1100, ui.Rect{X: del.X, Y: del.Y, W: 34, H: del.H}, theme.For(false, ui.Hex("#5B5BD6")).Blocked(), 40) {
		t.Errorf("the icon of Delete is not red")
	}
}

// paintedAnything reports whether the rectangle holds pixels that differ from the one at its corner.
func paintedAnything(img *image.RGBA, width float32, r ui.Rect) bool {
	scale := float32(img.Bounds().Dx()) / width
	base := img.RGBAAt(int(r.X*scale), int(r.Y*scale))
	for y := int(r.Y * scale); y < int((r.Y+r.H)*scale); y++ {
		for x := int(r.X * scale); x < int((r.X+r.W)*scale); x++ {
			p := img.RGBAAt(x, y)
			if absInt(int(p.R)-int(base.R))+absInt(int(p.G)-int(base.G))+absInt(int(p.B)-int(base.B)) > 60 {
				return true
			}
		}
	}
	return false
}
