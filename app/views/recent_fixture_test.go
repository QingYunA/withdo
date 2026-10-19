package views

import (
	"fmt"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// recentWords are what the assistant writes about the items of data.DemoRecent, by repository and
// number, in English and in Chinese.
var recentWords = []struct {
	repo    string
	number  int
	cat     data.RecentCategory
	en, zh  string
	topic   string
	topicZh string
}{
	{repo: "QingYunA/adhd-todo", number: 41, cat: data.RecentFixed, en: "Two-digit ledger totals fit", zh: "周报总数是两位数时，不再被截成省略号"},
	{repo: "QingYunA/adhd-todo", number: 42, cat: data.RecentFixed, en: "A flaky settings test stops failing", zh: "设置页取模型列表的测试不再偶尔失败"},
	{repo: "QingYunA/adhd-todo", number: 34, cat: data.RecentNew, en: "Summary plays quiet sounds while shipping", zh: "总结页出货时有轻微的木盒音效，设置里能关"},
	{repo: "QingYunA/adhd-todo", number: 35, cat: data.RecentNew, en: "The scene seats six, four agents get figures", zh: "场景能站 6 个人，四个 Agent 有了自己的小人"},
	{repo: "QingYunA/adhd-todo", number: 29, cat: data.RecentNew, en: "Settings copies the setup text for any agent", zh: "设置里能复制接入说明给任何 Agent"},
	{repo: "QingYunA/adhd-todo", number: 12, cat: data.RecentNew, en: "The native UI is in", zh: "原生界面上线：看板、列表、详情"},
	{repo: "QingYunA/adhd-todo", number: 14, cat: data.RecentNew, en: "The MCP server runs on the Go core", zh: "MCP 服务迁到了 Go 核心上"},
	{repo: "QingYunA/adhd-todo", number: 43, cat: data.RecentOther, en: "The recent changes page is specified", zh: "最近变化页有了设计稿", topic: "docs", topicZh: "文档"},
	{repo: "QingYunA/adhd-todo", number: 44, cat: data.RecentOther, en: "The version is 0.4.0", zh: "版本号升到 0.4.0", topic: "version", topicZh: "版本号"},
	{repo: "QingYunA/adhd-todo", number: 45, cat: data.RecentOther, en: "Summary screenshots are fresh", zh: "总结页截图已更新", topic: "screenshots", topicZh: "截图"},
	{repo: "QingYunA/adhd-todo", number: 39, en: "Work and Life get pixel icons", zh: "工作和生活分类有了像素图标"},
	{repo: "QingYunA/adhd-todo", number: 40, en: "The MCP server moves to 0.5", zh: "MCP 服务升到 0.5"},
	{repo: "QingYunA/adhd-todo", number: 58, en: "Export fails for tasks with pictures", zh: "导出 Markdown 时，带图片的任务会出错"},
	{repo: "QingYunA/adhd-todo", number: 56, en: "The ledger total shows an ellipsis", zh: "周报总数显示成省略号"},
	{repo: "QingYunA/picpocket", number: 44, cat: data.RecentNew, en: "A new worktree maps back to its repository", zh: "新建分支时自动把它的 worktree 映射回主仓库"},
	{repo: "QingYunA/picpocket", number: 43, cat: data.RecentFixed, en: "The tray no longer misses focus at launch", zh: "启动时托盘不再偶尔抢不到焦点"},
}

func isChinese(language string) bool { return language == "Simplified Chinese" }

// recentNoteFor is what the assistant says about an item, in a language.
func recentNoteFor(language, repo string, number int) (data.RecentNote, bool) {
	for _, w := range recentWords {
		if w.repo == repo && w.number == number {
			n := data.RecentNote{Sentence: w.en, Category: w.cat, Topic: w.topic}
			if isChinese(language) {
				n.Sentence, n.Topic = w.zh, w.topicZh
			}
			return n, true
		}
	}
	return data.RecentNote{}, false
}

// recentAssistant is an assistant with a model that writes the words above.
func recentAssistant(ready bool) *data.FakeAssistant {
	return &data.FakeAssistant{
		Ready: ready,
		RecentScript: func(req data.RecentDescribeRequest) (map[int]data.RecentNote, error) {
			out := map[int]data.RecentNote{}
			for _, i := range req.Items {
				if n, ok := recentNoteFor(req.Language, req.Repo, i.Number); ok {
					out[i.Number] = n
				}
			}
			return out, nil
		},
		RecentLinesScript: func(req data.RecentLinesRequest) (string, error) {
			switch {
			case req.Repo == "QingYunA/picpocket" && isChinese(req.Language):
				return "开始监视 worktree 里的新分支；修好了启动时托盘偶尔抢不到焦点。", nil
			case req.Repo == "QingYunA/picpocket":
				return "It now watches new branches in worktrees, and the tray no longer misses focus at launch.", nil
			case isChinese(req.Language):
				return "总结页有了出货音效和四个新角色，场景换上了 Kenney 的草地；修了周报总数被截断和一个偶发失败的测试。分类图标和 MCP 升级还在做。", nil
			}
			return "Summary gained shipping sounds and four new characters, and its scene stands on Kenney's grass; the cut ledger total and a flaky test are fixed. Category icons and the MCP upgrade are still in progress.", nil
		},
		RecentExplainScript: func(req data.RecentExplainRequest) (data.RecentExplanation, error) {
			if isChinese(req.Language) {
				return data.RecentExplanation{
					What:  "MCP 服务的工具搬到 Go 核心上，Claude Code 的 MCP 客户端照常能连。",
					Why:   "核心已经是 Go，MCP 服务不该再单独维护一套实现。",
					Where: "MCP 服务本身（internal/mcpserver）和它的测试。",
				}, nil
			}
			return data.RecentExplanation{
				What:  "The MCP server's tools now run on the Go core, and Claude Code's MCP client connects as before.",
				Why:   "The core is Go already, so the MCP server no longer needs an implementation of its own.",
				Where: "The MCP server itself (internal/mcpserver) and its tests.",
			}, nil
		},
	}
}

// recentRig is the Recent changes page over the seeded fake: the demo week in three repositories, the
// switch on, and an assistant that writes a sentence for each item.
type recentRig struct {
	app   *App
	store *data.Fake
	read  *data.FakeRecent
	asst  *data.FakeAssistant
	rec   *recorder
	tt    *ui.Tester
	m     *Main
}

// recentApp is the app with the page switched on over the demo week, and an assistant, with or
// without a model.
func recentApp(t *testing.T, model bool) (*App, *data.Fake, *data.FakeRecent, *data.FakeAssistant, *recorder) {
	t.Helper()
	app, store, rec := newTestApp(t)
	read, asst := data.DemoRecent(fixedNow()), recentAssistant(model)
	app.Assistant, app.Recent = asst, read
	app.GitHub = data.NewFakeGitHub(data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow()})
	app.Prefs = &data.MemoryPrefs{}
	app.SetRecentChangesOn(true)
	app.Refresh()
	return app, store, read, asst, rec
}

// newRecentRig opens a tester on the app. With model false the assistant has no model.
func newRecentRig(t *testing.T, model bool, mod ...func(*recentRig)) *recentRig {
	t.Helper()
	app, store, read, asst, rec := recentApp(t, model)
	r := &recentRig{app: app, store: store, rec: rec, read: read, asst: asst}
	for _, f := range mod {
		f(r)
	}
	r.tt, r.m, _ = newTesterFor(app, 1100)
	r.tt.SetSize(1100, 900)
	return r
}

// open shows the page and waits for GitHub's answer and for the assistant's work.
func (r *recentRig) open(t *testing.T) {
	t.Helper()
	r.m.goTo(nav{screen: screenRecent})
	r.tt.Frame()
	r.settle(t)
}

func (r *recentRig) settle(t *testing.T) { t.Helper(); settleRecent(t, r.tt, r.rec) }

// settleRecent runs what was posted to the UI thread and draws, until nothing is on its way.
func settleRecent(t *testing.T, tt *ui.Tester, rec *recorder) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for quiet := 0; quiet < 5 && time.Now().Before(deadline); {
		rec.mu.Lock()
		fns := rec.posted
		rec.posted = nil
		rec.mu.Unlock()
		for _, fn := range fns {
			fn()
		}
		tt.Frame()
		if len(fns) == 0 {
			quiet++
			time.Sleep(15 * time.Millisecond)
		} else {
			quiet = 0
		}
	}
}

func (r *recentRig) click(t *testing.T, text string) {
	t.Helper()
	if err := r.tt.Click(text); err != nil {
		t.Fatalf("click %q: %v in %q", text, err, r.tt.Texts())
	}
	r.tt.Frame()
}

// has fails the test for each text that is not on the page.
func (r *recentRig) has(t *testing.T, texts ...string) {
	t.Helper()
	for _, want := range texts {
		if !r.tt.HasText(want) {
			t.Errorf("missing %q in %q", want, r.tt.Texts())
		}
	}
}

// lacks fails the test for each text that is on the page.
func (r *recentRig) lacks(t *testing.T, texts ...string) {
	t.Helper()
	for _, bad := range texts {
		if r.tt.HasText(bad) {
			t.Errorf("unexpected %q in %q", bad, r.tt.Texts())
		}
	}
}

func (r *recentRig) fetches() string { return fmt.Sprint(r.read.Fetched()) }
