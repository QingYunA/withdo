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

func TestChineseNowScreen(t *testing.T) {
	tt, _, _ := newChineseTester(t, 1100)
	has(t, tt,
		"现在", "10月14日星期三",
		"需要你处理", "Agent 进行中", "你的任务",
		"Agent 在等待：“Needs a decision on the activation policy”",
		"你设置了下午2:30的提醒", "PR #12 等你审查",
		"打开 PR", "完成", "打开",
		"3天前", "2分钟前",
		"今天", "明天", "周五",
		"收件箱", "项目", "设置",
	)
	if !above(t, tt, "需要你处理", "Agent 进行中") || !above(t, tt, "Agent 进行中", "你的任务") {
		t.Error("sections are out of order")
	}
	hasNot(t, tt, "Needs you", "Settings", "Inbox")
}

func TestChineseSectionHeadersAreNotSpacedOrCapitalized(t *testing.T) {
	tt, _, _ := newChineseTester(t, 1100)
	r, ok := tt.Find("需要你处理")
	if !ok {
		t.Fatalf("no header in %q", tt.Texts())
	}
	// Letter spacing on each of 5 characters would add to the 13 pt glyphs' width.
	if w := r.W; w > 5*13+0.5 {
		t.Errorf("the header is %v pt wide: Han characters were spaced", w)
	}
}

func TestChineseAllClearAndAsk(t *testing.T) {
	app, _ := newEmptyApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	inChinese(tt)
	has(t, tt, "全部处理完了。", "Agent 会通过 adhd-todo 的 MCP 服务器在这里汇报进度。", "如何连接",
		"告诉我要记住什么，比如“下午4点提醒我审查 PR #12”", "Agent 从代码仓库汇报进度后，项目会出现在这里。")
}

func TestLanguageFollowsTheWindowLive(t *testing.T) {
	tt, _, _ := newMainTester(t, 1100)
	has(t, tt, "Needs you")
	inChinese(tt)
	has(t, tt, "需要你处理")
	hasNot(t, tt, "Needs you")
	tt.SetLocale("en")
	has(t, tt, "Needs you")
}

func TestChineseProjectList(t *testing.T) {
	tt, m, _ := newChineseTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 1})
	tt.SetSize(1100, 1600) // room for the Done group
	tt.Frame()
	has(t, tt, "已阻塞", "待审查", "进行中", "待办", "+ 添加任务", "已完成", "显示全部 5 项 →", "待规划 2", "— 显示", "停滞3天")
	if err := tt.Click("+ 添加任务"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "任务标题")
	tt.Key(0, ui.KeyEscape)
	if err := tt.Click("显示待规划 2"); err != nil {
		t.Fatal(err)
	}
	has(t, tt, "— 隐藏", "已完成", "待规划")
}

func TestChineseBoardAndInbox(t *testing.T) {
	tt, m, _ := newChineseTester(t, 1100)
	m.goTo(nav{screen: screenProject, projectID: 2})
	m.board[2] = true
	tt.Frame()
	has(t, tt, "列表", "看板", "待办")
	m.goTo(nav{screen: screenInbox})
	tt.Frame()
	has(t, tt, "收件箱", "没有归属项目的任务")
	app, p := emptyProjectApp(t)
	tt2, m2, _ := newTesterFor(app, 1100)
	inChinese(tt2)
	m2.goTo(nav{screen: screenProject, projectID: p.ID})
	tt2.Frame()
	has(t, tt2, "还没有任务。", "编程 Agent 会通过 adhd-todo 的 MCP 服务器把任务添加到这里，你也可以自己添加。")
}

func TestChineseInspector(t *testing.T) {
	tt, m, _ := newChineseTester(t, 1100)
	m.OpenTask(3)
	tt.Frame()
	has(t, tt, "状态", "优先级", "里程碑", "截止", "提醒", "分支", "PR", "动态", "已阻塞", "P0  紧急",
		"下一步：Needs a decision on the activation policy")
}

func TestChineseSettingsAndLanguageChoice(t *testing.T) {
	app, _, rec := newTestApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	inChinese(tt)
	m.OpenSettings()
	tt.Frame()
	has(t, tt, "设置", "连接 Claude Code", "在终端里运行一次，之后 Agent 的进度就会汇报到这里。", "数据文件夹", "语言",
		"跟随系统", "English", "简体中文", "adhd-todo 0.2.0  ·  开源，MIT 许可证", "完成")
	if err := tt.Click("English"); err != nil {
		t.Fatal(err)
	}
	if rec.language != "en" {
		t.Errorf("chose %q", rec.language)
	}
	if err := tt.Click("跟随系统"); err != nil {
		t.Fatal(err)
	}
	if rec.language != "" {
		t.Errorf("chose %q after System", rec.language)
	}
}

func TestChineseModelsTab(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	inChinese(r.tt)
	r.m.openModels()
	r.tt.Frame()
	has(t, r.tt, "通用", "模型", "Base URL", "API 密钥", "需要 API 密钥", "使用这个", "测试连接", "自定义")
	hasNot(t, r.tt, "Custom")
	if err := r.tt.Click("Magpie"); err != nil {
		t.Fatal(err)
	}
	has(t, r.tt, "使用本机的 Magpie 网关 127.0.0.1:3425")
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

func TestChineseNoModelHintAndInboxNotice(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	inChinese(r.tt)
	r.ask("浇花")
	has(t, r.tt, "已添加到收件箱")
	r.now = r.now.Add(2 * time.Second)
	r.tt.Frame()
	has(t, r.tt, "连接模型后，我才能听懂你的话 —", "去设置")
}
