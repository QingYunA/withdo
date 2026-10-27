package views

import (
	"errors"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

// These tests draw the screens in Simplified Chinese, with the regional formats of a reader who
// keeps 12 hours, and look for the words a reader would see.

// inChinese shows the tester's window in Simplified Chinese.
func inChinese(tt *ui.Tester) {
	tt.SetLocale("zh-Hans")
	tt.SetRegion(i18n.Region{Locale: "zh-CN", HourCycle: 12})
}

func newChineseTester(t *testing.T, width int) (*ui.Tester, *Main, *App) {
	t.Helper()
	tt, m, app := newMainTester(t, width)
	inChinese(tt)
	return tt, m, app
}

func TestChineseMineScreen(t *testing.T) {
	tt, _, _ := newChineseTester(t, inlineWidth)
	has(t, tt,
		"10月14日 星期三",
		"Agent 交过来的", "今天", "接下来",
		"等你回复：Needs a decision on the activation policy",
		"下午2:30", "PR #12 待审查",
		"打开 PR", "回复", "标为完成",
		"3天前", "25分钟前",
		"明天", "周五",
		"日历", "项目", "设置",
	)
	if !above(t, tt, "Agent 交过来的", "接下来") {
		t.Error("sections are out of order")
	}
	hasNot(t, tt, "From your agents", "Settings", "Inbox", "收件箱", "我的")
}

func TestChineseSectionHeadersAreNotSpacedOrCapitalized(t *testing.T) {
	tt, _, _ := newChineseTester(t, 1100)
	r, ok := tt.Find("接下来")
	if !ok {
		t.Fatalf("no header in %q", tt.Texts())
	}
	// Letter spacing on each of 3 characters would add to the 13 pt glyphs' width.
	if w := r.W; w > 3*13+0.5 {
		t.Errorf("the header is %v pt wide: Han characters were spaced", w)
	}
}

func TestChineseAllClearAndAsk(t *testing.T) {
	app, _ := newEmptyApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	inChinese(tt)
	has(t, tt, "现在没有要你处理的事", "Agent 们在干活，有动静会出现在这里", "如何连接",
		"明天 9 点给房东打电话，再让 Claude 修一下登录的 bug", "Agent 从代码仓库汇报进度后，项目会出现在这里。")
}

func TestLanguageFollowsTheWindowLive(t *testing.T) {
	tt, _, _ := newMainTester(t, inlineWidth)
	has(t, tt, "From your agents")
	inChinese(tt)
	has(t, tt, "Agent 交过来的")
	hasNot(t, tt, "From your agents")
	tt.SetLocale("en")
	has(t, tt, "From your agents")
}

func TestChineseProjectList(t *testing.T) {
	tt, m, _ := newChineseTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.SetSize(1100, 1600) // room for the Done group
	tt.Frame()
	has(t, tt, "已阻塞", "待审查", "进行中", "待办", "已完成", "我的", "待规划 2", "— 显示", "停滞3天",
		"添加任务，或者问我任何事")
	if err := tt.Click("显示待规划 2"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "— 隐藏", "已完成", "待规划")
}

func TestChineseBoardAndEmptyProject(t *testing.T) {
	tt, m, _ := newChineseTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 2})
	m.board[2] = true
	tt.Frame()
	has(t, tt, "列表", "看板", "待办")
	app, p := emptyProjectApp(t)
	tt2, m2, _ := newTesterFor(app, 1100)
	inChinese(tt2)
	m2.goTo(nav{screen: screenProject, projectID: p.ID})
	tt2.Frame()
	has(t, tt2, "还没有任务。", "编程 Agent 会通过 Withdo 的 MCP 服务把任务添加到这里，你也可以自己添加。")
}

func TestChineseInspector(t *testing.T) {
	tt, m, _ := newChineseTester(t, 1100)
	m.OpenTask(3)
	tt.Frame()
	has(t, tt, "状态", "优先级", "里程碑", "截止", "提醒", "分支", "PR", "动态", "已阻塞", "P0  紧急",
		"下一步：Needs a decision on the activation policy")
}

func TestChineseSettingsAndLanguageChoice(t *testing.T) {
	r := newSettingsRig(t)
	inChinese(r.tt)
	r.tt.Frame()
	has(t, r.tt, "账号", "通用", "Agent", "高级", "语言", "跟随系统", "已归档的项目", "没有已归档的项目。在侧边栏中右键点按项目，可以归档它。",
		"数据文件夹", "在访达中显示", "Withdo 0.2.0", "开源，MIT 许可证")
	if r.title != "通用" {
		t.Errorf("title %q", r.title)
	}
	if err := r.tt.Click("跟随系统"); err != nil {
		t.Fatal(err)
	}
	if err := r.tt.Click("English"); err != nil {
		t.Fatal(err)
	}
	if r.rec.language != "en" {
		t.Errorf("chose %q", r.rec.language)
	}
}

func TestChineseAssistantPane(t *testing.T) {
	r := newSettingsRig(t)
	inChinese(r.tt)
	r.setUp(t, "anthropic", "deepseek")
	r.pane(t, "高级")
	has(t, r.tt, "更换", "用这个", "测试", "还没有测试", "claude-haiku-5-5")
	hasFold(t, r.tt, "当前在用")
	hasFold(t, r.tt, "也配好了")
	hasNot(t, r.tt, "In use", "Not tested yet", "Change")
	mustNil(t, r.tt.Click("测试"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "已连接 · 640 ms")
	mustNil(t, r.tt.Click("更换"))
	mustNil(t, r.tt.Click("claude-haiku-5-5"))
	has(t, r.tt, "其他…", "API 密钥", "保存并测试")
}

func TestChineseAgentsPane(t *testing.T) {
	r := newSettingsRig(t)
	inChinese(r.tt)
	r.pane(t, "Agent")
	has(t, r.tt, "编程 Agent", "Claude Code", "拷贝")
	hasNot(t, r.tt, "连接 Claude Code")
}

func TestChineseConversation(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true, Script: scripted(reminderReply(), nil)}
	r := newAssistRig(t, fa)
	inChinese(r.tt)
	r.ask("明天九点提醒我开会")
	r.answer(t)
	has(t, r.tt, "Team meeting", "明天上午9:00 · 提醒", "撤销")
	if err := r.tt.Click("撤销"); err != nil {
		t.Fatal(err)
	}
	has(t, r.tt, "已撤销")
}

func TestChineseQuestionAndErrors(t *testing.T) {
	fa := &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{Question: &data.Question{
		Text: "Which one?", Options: []string{"A", "B"},
	}}, nil)}
	r := newAssistRig(t, fa)
	inChinese(r.tt)
	r.ask("release notes are done")
	r.answer(t)
	has(t, r.tt, "Which one?", "输入回答，或在下方选择")

	bad := newAssistRig(t, &data.FakeAssistant{Ready: true, Script: scripted(data.Reply{},
		&data.ProblemError{Problem: data.ProblemUnreachable, Text: "Couldn't reach the model.", Err: errors.New("refused")})})
	inChinese(bad.tt)
	bad.ask("do it")
	bad.answer(t)
	has(t, bad.tt, "无法连接模型。请检查地址是否正确、服务是否在运行。", "重试")
	hasNot(t, bad.tt, "Couldn't reach the model.")
}

func TestChineseNoModelSetUpLineAndInboxNotice(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	inChinese(r.tt)
	hasNot(t, r.tt, "设置助手")
	r.tt.Key(ui.Cmd, ui.KeyK)
	has(t, r.tt, "设置助手")
	r.tt.Type("x")
	r.tt.Key(0, ui.KeyBackspace)
	r.ask("浇花")
	has(t, r.tt, "已添加到没定时间")
	r.now = r.now.Add(2 * time.Second)
	r.tt.Frame()
	hasNot(t, r.tt, "已添加到没定时间")
}

func TestChineseComposerPlaceholders(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	inChinese(r.tt)
	r.m.OpenTask(5)
	r.tt.Frame()
	has(t, r.tt, "明天 9 点给房东打电话，再让 Claude 修一下登录的 bug", "说说要怎么改这个任务")
	hasNot(t, r.tt, "助手 ·")
}
