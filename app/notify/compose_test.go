package notify

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/locales"
	"github.com/egoist/mygo"
)

var (
	en = locales.In("en")
	zh = locales.In("zh-Hans")
)

func TestEventsAreWordedPerTheSpec(t *testing.T) {
	handed := data.Event{Kind: data.EventHandedBack, TaskID: 7, Title: "Native UI with mygo", ProjectName: "adhd-todo", Agent: "claude-code", Detail: "All tests pass"}
	withPR := handed
	withPR.PRNumber = 12
	stuck := data.Event{Kind: data.EventStuck, TaskID: 8, Title: "Fix tray race", ProjectName: "adhd-todo", Agent: "codex", Detail: "Which activation policy?"}
	checks := data.Event{Kind: data.EventChecksFailing, TaskID: 7, Title: "Native UI with mygo", ProjectName: "adhd-todo", PRNumber: 12}
	merged := data.Event{Kind: data.EventMerged, TaskID: 9, Title: "Port the MCP server", PRNumber: 14}
	tests := []struct {
		name string
		say  locales.Messages
		in   data.Event
		want mygo.NotificationOptions
	}{
		{"handed back, no PR", en, handed, mygo.NotificationOptions{ID: "task/7", Title: "Claude Code finished: Native UI with mygo", Subtitle: "adhd-todo", Body: "All tests pass", Group: "project/adhd-todo"}},
		{"handed back with a PR", en, withPR, mygo.NotificationOptions{ID: "task/7", Title: "Claude Code finished: Native UI with mygo", Subtitle: "adhd-todo", Body: "PR #12 is ready for your review", Group: "project/adhd-todo"}},
		{"stuck", en, stuck, mygo.NotificationOptions{ID: "task/8", Title: "Codex is stuck: Fix tray race", Subtitle: "adhd-todo", Body: "Which activation policy?", Group: "project/adhd-todo"}},
		{"checks failing", en, checks, mygo.NotificationOptions{ID: "task/7", Title: "Checks are failing on PR #12: Native UI with mygo", Subtitle: "adhd-todo", Body: "Open the PR to see which one failed", Group: "project/adhd-todo"}},
		{"merged is silent, no project is one stack", en, merged, mygo.NotificationOptions{ID: "task/9", Title: "PR #14 merged: Port the MCP server", Subtitle: "Today", Body: "The task is done", Silent: true, Group: "inbox"}},
		{"handed back, zh", zh, withPR, mygo.NotificationOptions{ID: "task/7", Title: "Claude Code 做完了：Native UI with mygo", Subtitle: "adhd-todo", Body: "PR #12 等你审查", Group: "project/adhd-todo"}},
		{"stuck, zh", zh, stuck, mygo.NotificationOptions{ID: "task/8", Title: "Codex 卡住了：Fix tray race", Subtitle: "adhd-todo", Body: "Which activation policy?", Group: "project/adhd-todo"}},
		{"checks failing, zh", zh, checks, mygo.NotificationOptions{ID: "task/7", Title: "PR #12 检查没过：Native UI with mygo", Subtitle: "adhd-todo", Body: "打开 PR 看看哪一项失败", Group: "project/adhd-todo"}},
		{"merged, zh", zh, merged, mygo.NotificationOptions{ID: "task/9", Title: "PR #14 已合并：Port the MCP server", Subtitle: "今天", Body: "任务已自动完成", Silent: true, Group: "inbox"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Event(tt.say, tt.in); got != tt.want {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestAReminderIsWordedAsBefore(t *testing.T) {
	got := Reminder(en, data.Reminder{TaskID: 3, Title: "Pay rent"})
	if want := (mygo.NotificationOptions{ID: "task/3", Title: "Reminder: Pay rent", Body: "Today"}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	got = Reminder(zh, data.Reminder{TaskID: 3, Title: "交房租", ProjectName: "家"})
	if want := (mygo.NotificationOptions{ID: "task/3", Title: "提醒：交房租", Body: "家"}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestTheSummaryCountsEachKindThatOccurred(t *testing.T) {
	c := Counts{HandedBack: 2, Stuck: 1}
	if got, want := Summary(en, c), (mygo.NotificationOptions{ID: "summary", Title: "3 things need you", Body: "2 finished, 1 stuck · open Today"}); got != want {
		t.Errorf("en: got %+v, want %+v", got, want)
	}
	if got, want := Summary(zh, c), (mygo.NotificationOptions{ID: "summary", Title: "3 件事等你", Body: "2 个做完了，1 个卡住了 · 点开去「今天」"}); got != want {
		t.Errorf("zh: got %+v, want %+v", got, want)
	}
	all := Summary(en, Counts{HandedBack: 1, Stuck: 1, ChecksFailing: 1, Merged: 2})
	if all.Body != "1 finished, 1 stuck, 1 with failing checks, 2 merged · open Today" || all.Silent {
		t.Errorf("all kinds = %+v", all)
	}
	if !Summary(en, Counts{Merged: 3}).Silent {
		t.Error("a summary of merges only should be silent")
	}
}

func TestTaskIDsRoundTrip(t *testing.T) {
	if id, ok := ParseTaskID(TaskID(42)); !ok || id != 42 {
		t.Errorf("ParseTaskID(TaskID(42)) = %d, %v", id, ok)
	}
	for _, bad := range []string{"", SummaryID, "task/0", "task/x"} {
		if _, ok := ParseTaskID(bad); ok {
			t.Errorf("ParseTaskID(%q) should fail", bad)
		}
	}
}
