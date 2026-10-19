//go:build promo

package views

import (
	"fmt"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// The made-up week of the promo video (docs/design/promo.md, The data): nothing here comes from a
// real database. Friday 16 October 2026 is today, at 17:30 (the dusk sky); the week is Monday to
// Friday, with Saturday and Sunday still empty.

// promoBase is the app's clock when the video's window segment begins: Friday 17:30.
var promoBase = time.Date(2026, 10, 16, 17, 30, 0, 0, time.Local)

// promoTask is one finished task: the weekday (0 = Monday), when, who finished it, in which
// project, what kind of good it is, whether its pull request merged, and its title in both languages.
type promoTask struct {
	day    int
	at     string // "15:04"
	who    string // key of promoWho
	proj   int    // index of promoProjects
	kind   string
	merged bool
	en, zh string
}

// promoWho maps who finished a task to the name an agent reports under; "" is you.
var promoWho = map[string]string{
	"you": "", "claude": "claude-code", "codex": "codex", "gemini": "gemini-cli",
	"cursor": "cursor", "copilot": "GitHub Copilot", "opencode": "opencode",
}

var promoProjects = []string{"adhd-todo", "picpocket", "blog", "dotfiles"}

// promoTasks are the 48 tasks of the week. Today (Friday) has 16: you 3, Claude Code 4, Codex 3,
// Gemini CLI 2, Cursor 2, GitHub Copilot 1, OpenCode 1, with 7 pull requests merged (Claude Code 3,
// Codex 2, Gemini CLI 1, Cursor 1). Monday 8, Tuesday 6, Wednesday 10, Thursday 8.
var promoTasks = []promoTask{
	// Monday: you 2, Claude Code 3, Codex 2, Gemini CLI 1
	{0, "09:20", "you", 2, "book", false, "Outline the Monday post", "列好周一文章的提纲"},
	{0, "09:50", "claude", 0, "gem", true, "Add the Today | This week switch", "加上「今天 | 本周」切换"},
	{0, "10:30", "codex", 3, "gear", true, "Split the zsh config into modules", "把 zsh 配置拆成模块"},
	{0, "11:10", "gemini", 0, "flask", false, "Test the date paging", "给日期翻页补测试"},
	{0, "12:00", "claude", 1, "jar", true, "Fix the blurry thumbnails on 2x screens", "修 2x 屏上缩略图发虚"},
	{0, "13:40", "you", 0, "mail", false, "Answer the beta tester's question", "回内测用户的问题"},
	{0, "14:30", "codex", 0, "gem", true, "Add the reminder watcher", "加提醒监听"},
	{0, "16:05", "claude", 0, "gear", false, "Move the store behind an interface", "把存储藏到接口后面"},
	// Tuesday: you 1, Claude Code 2, Codex 1, Cursor 1, GitHub Copilot 1
	{1, "09:40", "you", 2, "scroll", false, "Plan the autumn posts", "排好秋天的文章"},
	{1, "10:15", "claude", 0, "gem", true, "Add the sidebar's Agents section", "加侧边栏的 Agent 分组"},
	{1, "11:30", "cursor", 1, "gem", true, "Add a crop tool to the viewer", "给查看器加裁剪工具"},
	{1, "13:20", "codex", 1, "jar", false, "Fix the duplicate import", "修重复导入"},
	{1, "14:50", "copilot", 3, "book", false, "Document the install script", "写安装脚本说明"},
	{1, "16:40", "claude", 2, "book", false, "Draft the changelog", "起草更新日志"},
	// Wednesday: you 2, Claude Code 2, Codex 2, Gemini CLI 2, Cursor 1, GitHub Copilot 1
	{2, "09:10", "you", 0, "coin", false, "Pay the hosting bill", "付托管费"},
	{2, "09:55", "claude", 0, "gem", true, "Light the ledger rows one by one", "让账本逐行亮起"},
	{2, "10:40", "gemini", 1, "jar", true, "Fix the crash on empty albums", "修空相册崩溃"},
	{2, "11:25", "codex", 0, "gear", true, "Migrate the schema to version 10", "把库表迁到第 10 版"},
	{2, "12:10", "copilot", 2, "crate", false, "Tidy the old drafts", "清理旧草稿"},
	{2, "13:15", "cursor", 0, "flask", false, "Cover the board with tests", "给看板补测试"},
	{2, "14:05", "you", 0, "mail", false, "Reply to the design feedback", "回复设计反馈"},
	{2, "15:00", "claude", 1, "gear", true, "Replace the thumbnail cache", "换掉缩略图缓存"},
	{2, "15:45", "gemini", 0, "scroll", false, "Research how others sound the done", "调研别人怎么做完成音"},
	{2, "16:50", "codex", 3, "gem", false, "Add a bootstrap for new Macs", "给新 Mac 加引导脚本"},
	// Thursday: you 1, Claude Code 2, Codex 1, Gemini CLI 1, Cursor 1, GitHub Copilot 1, OpenCode 1
	{3, "09:35", "you", 2, "book", false, "Write the Thursday post", "写周四的文章"},
	{3, "10:20", "claude", 0, "gem", true, "Count the ledger up", "让账本数字滚动"},
	{3, "11:05", "codex", 1, "flask", true, "Cover the importer with tests", "给导入器补测试"},
	{3, "11:50", "opencode", 3, "gear", false, "Rewrite the Neovim keymaps", "重写 Neovim 键位"},
	{3, "13:30", "gemini", 0, "jar", true, "Fix the tray race on launch", "修启动时的托盘竞态"},
	{3, "14:20", "cursor", 0, "gem", true, "Add drag and drop to the board", "给看板加拖拽"},
	{3, "15:30", "copilot", 1, "mail", false, "Answer the issue about HEIC", "回关于 HEIC 的 issue"},
	{3, "16:30", "claude", 0, "scroll", false, "Plan the end-of-day reminder", "规划收工提醒"},
	// Friday (today): you 3, Claude Code 4, Codex 3, Gemini CLI 2, Cursor 2, GitHub Copilot 1, OpenCode 1
	{4, "09:05", "you", 2, "book", false, "Write the Friday changelog post", "写周五的更新日志"},
	{4, "09:40", "claude", 0, "gem", true, "Add the Summary page's export sheet", "加上总结页的导出面板"},
	{4, "10:10", "codex", 3, "gear", true, "Move the shell setup to one bootstrap", "把 shell 配置合成一个引导脚本"},
	{4, "10:35", "gemini", 0, "jar", true, "Fix the empty-state crash", "修空状态下的崩溃"},
	{4, "11:00", "cursor", 0, "gem", true, "Add keyboard shortcuts to the board", "给看板加键盘快捷键"},
	{4, "11:20", "copilot", 1, "book", false, "Write the Windows setup notes", "写 Windows 安装说明"},
	{4, "11:45", "claude", 0, "gear", true, "Port the MCP server to Go", "把 MCP 服务移植到 Go"},
	{4, "12:30", "you", 0, "mail", false, "Reply to the design feedback", "回复设计反馈"},
	{4, "13:05", "codex", 1, "flask", false, "Cover the thumbnail cache with tests", "给缩略图缓存补测试"},
	{4, "13:30", "gemini", 0, "flask", false, "Cover the export sheet with tests", "给导出面板补测试"},
	{4, "14:10", "claude", 1, "jar", true, "Fix the HEIC export colour shift", "修 HEIC 导出的偏色"},
	{4, "14:40", "cursor", 0, "gear", false, "Tidy the sidebar spacing", "整理侧边栏间距"},
	{4, "15:15", "you", 2, "coin", false, "Pay the domain renewal", "付域名续费"},
	{4, "15:50", "codex", 0, "gem", true, "Port the reminder watcher", "移植提醒监听"},
	{4, "16:20", "claude", 2, "scroll", false, "Plan the sync engine rework", "规划同步引擎的重构"},
	{4, "16:55", "opencode", 3, "crate", false, "Clean up the old Neovim plugins", "清理旧的 Neovim 插件"},
}

const promoDayParagraphEN = "Sixteen finished today. Claude Code, Codex, Gemini CLI and Cursor merged seven pull requests; the sync plan is still open."
const promoDayParagraphZH = "今天收了 16 件。Claude Code、Codex、Gemini CLI 和 Cursor 合并了 7 个 PR，同步引擎的方案还在等你。"
const promoWeekParagraphEN = "Forty-eight finished this week, Friday the busiest with sixteen. Claude Code led with thirteen; Monday to Thursday stayed steady."
const promoWeekParagraphZH = "这周收了 48 件，周五最多，16 件。Claude Code 做了 13 件，周一到周四节奏平稳。"

// promoApp is the app of the video: the Fake store with the week's tasks, a few open ones for the
// Mine page, and the stored paragraphs of today and this week, in the language of zh. now is the
// app's clock.
func promoApp(t *testing.T, now func() time.Time, zh bool) (*App, error) {
	t.Helper()
	store := data.NewEmptyFake(now)
	var projects []int
	for _, name := range promoProjects {
		projects = append(projects, store.AddProject(name, "QingYunA/"+name).ID)
	}
	say := func(en, zhText string) string {
		if zh {
			return zhText
		}
		return en
	}
	y, mo, d := promoBase.Date()
	monday := d - int(promoBase.Weekday()-time.Monday)
	kinds := map[int]string{}
	pr := 40
	for i, pt := range promoTasks {
		id := 300 + i
		clock, err := time.ParseInLocation("15:04", pt.at, time.Local)
		if err != nil {
			return nil, err
		}
		at := time.Date(y, mo, monday+pt.day, clock.Hour(), clock.Minute(), 0, 0, time.Local)
		title := say(pt.en, pt.zh)
		task := data.Task{ID: id, ProjectID: data.IntPtr(projects[pt.proj]), Title: title, Status: data.Doing, Source: data.SourceUser, CreatedAt: at.Add(-26 * time.Hour)}
		if pt.who != "you" {
			task.Source = data.SourceAgent
			task.Branch = fmt.Sprintf("%s/task-%d", pt.who, id)
		}
		if pt.merged {
			pr++
			task.PRNumber = pr
			task.PRURL = fmt.Sprintf("https://github.com/QingYunA/%s/pull/%d", promoProjects[pt.proj], pr)
		}
		store.AddTask(task)
		if agent := promoWho[pt.who]; agent != "" {
			store.ReportAs(id, agent, title, at.Add(-time.Second))
		}
		if pt.merged {
			if err := store.SetPullRequest(id, data.PRMerged, data.PRChecksPassing, data.PRReviewApproved); err != nil {
				return nil, err
			}
		}
		store.CompleteAt(id, at)
		kinds[id] = pt.kind
	}
	promoOpenTasks(store, projects, zh, now())

	cfg := testConfig()
	cfg.AgentSetup = newAgentRig(t).env
	app := NewApp(store, &recorder{}, cfg)
	app.Now = now
	app.Assistant = &data.FakeAssistant{Ready: true}
	_, _ = store.PutSummary(data.DayPeriod(now(), 0), data.StoredSummary{Paragraph: say(promoDayParagraphEN, promoDayParagraphZH), Model: "fake-model", Kinds: kinds})
	_, _ = store.PutSummary(data.WeekPeriod(now(), 0, time.Monday), data.StoredSummary{Paragraph: say(promoWeekParagraphEN, promoWeekParagraphZH), Model: "fake-model", Kinds: kinds})
	app.Refresh()
	return app, nil
}

// promoOpenTasks are what is still open, for the Mine page the video starts on: an agent stuck, a
// pull request waiting for review, agents at work, and a few of your own.
func promoOpenTasks(store *data.Fake, projects []int, zh bool, now time.Time) {
	say := func(en, zhText string) string {
		if zh {
			return zhText
		}
		return en
	}
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	y, mo, d := now.Date()
	at := func(days, h int) *time.Time { t := time.Date(y, mo, d+days, h, 0, 0, 0, time.Local); return &t }

	add := func(t data.Task, agent string, st data.Status, said, next string, since time.Duration) {
		t.CreatedAt, t.UpdatedAt = ago(since+26*time.Hour), ago(since)
		store.AddTask(t)
		if agent != "" {
			store.AddReport(data.Report{TaskID: t.ID, Agent: agent, Status: st, Summary: said, NextStep: next, CreatedAt: ago(since)})
		}
	}
	add(data.Task{ID: 400, ProjectID: data.IntPtr(projects[0]), Title: say("Fix the tray race on macOS 27", "修 macOS 27 上的托盘竞态"),
		Status: data.Blocked, Priority: data.P0, Source: data.SourceAgent, Branch: "fix/tray"},
		"claude-code", data.Blocked, say("The tray icon shows before the app is ready", "托盘图标在应用就绪前就出现了"),
		say("Needs a decision on the activation policy", "需要你定激活策略"), 22*time.Minute)
	add(data.Task{ID: 401, ProjectID: data.IntPtr(projects[0]), Title: say("Move sync to a background queue", "把同步挪到后台队列"),
		Status: data.Review, Priority: data.P1, Source: data.SourceAgent, Branch: "feat/sync-queue", PRNumber: 61,
		PRURL: "https://github.com/QingYunA/adhd-todo/pull/61", PRState: data.PROpen, PRChecks: data.PRChecksPassing, PRReview: data.PRReviewRequired},
		"codex", data.Review, say("Queue is in; waiting for your review", "队列写好了，等你审查"), "", 9*time.Minute)
	add(data.Task{ID: 402, ProjectID: data.IntPtr(projects[1]), Title: say("Batch-export to HEIC", "批量导出 HEIC"),
		Status: data.Doing, Source: data.SourceAgent, Branch: "feat/heic"},
		"gemini-cli", data.Doing, say("Reading the export code", "在读导出代码"), "", 3*time.Minute)
	add(data.Task{ID: 403, ProjectID: data.IntPtr(projects[3]), Title: say("Try a new prompt for the shell", "试一套新的 shell 提示符"),
		Status: data.Doing, Source: data.SourceAgent, Branch: "feat/prompt"},
		"opencode", data.Doing, say("Comparing three prompts", "在比较三套提示符"), "", 5*time.Minute)
	add(data.Task{ID: 410, ProjectID: data.IntPtr(projects[0]), Title: say("Record the 30 s demo", "录 30 秒的演示"),
		Status: data.Todo, Priority: data.P1, DueAt: at(0, 19)}, "", "", "", "", 4*time.Hour)
	add(data.Task{ID: 411, ProjectID: data.IntPtr(projects[2]), Title: say("Write the release notes", "写发布说明"),
		Status: data.Todo, DueAt: at(3, 10)}, "", "", "", "", 20*time.Hour)
	add(data.Task{ID: 412, Title: say("Book the dentist", "预约牙医"), Status: data.Todo, Category: data.CategoryLife, DueAt: at(1, 9)}, "", "", "", "", 40*time.Hour)
	add(data.Task{ID: 413, Title: say("Reply to Mia about the logo", "回 Mia 关于 logo 的消息"), Status: data.Todo, Category: data.CategoryWork,
		RemindAt: at(0, 18)}, "", "", "", "", 3*time.Hour)
	add(data.Task{ID: 414, Title: say("Renew the passport", "办护照续期"), Status: data.Backlog, Category: data.CategoryLife}, "", "", "", "", 9*24*time.Hour)
}
