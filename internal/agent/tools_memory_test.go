package agent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

var memoryDay = time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)

// memRig is a rig whose assistant has a memory in a throwaway folder.
func memRig(t *testing.T, steps ...step) (*rig, *Memory) {
	t.Helper()
	r := newRig(t, "UTC", memoryDay, steps...)
	mem := newMem(t)
	r.agent.cfg.Memory = mem
	return r, mem
}

func systemOf(in []*schema.Message) string {
	for _, m := range in {
		if m.Role == schema.System {
			return m.Content
		}
	}
	return ""
}

func TestPromptHasNotesOnlyWhenThereAreSome(t *testing.T) {
	r, mem := memRig(t, say("ok"), say("ok"))
	r.ask("hi")
	sys := systemOf(r.model.sent(0))
	contains(t, "prompt", sys, "remember")
	if strings.Contains(sys, "User notes") {
		t.Errorf("a notes block without notes:\n%s", sys)
	}
	mustAdd(t, mem, "about", "叫我 Cyan")
	r.ask("hi again")
	sys = systemOf(r.model.sent(1))
	contains(t, "prompt", sys, "User notes (from the user's corrections and preferences; obey them over the default rules above):")
	contains(t, "prompt", sys, "- 叫我 Cyan")

	plain := newRig(t, "UTC", memoryDay, say("ok"))
	plain.ask("hi")
	if s := systemOf(plain.model.sent(0)); strings.Contains(s, "User notes") || strings.Contains(s, "remember") {
		t.Errorf("a prompt about memory without a memory:\n%s", s)
	}
}

func TestForgetThenRememberLeavesOneNoteWithReceiptsAndUndo(t *testing.T) {
	r, mem := memRig(t,
		call(toolRemember, m{"note": "健身归「工作」类", "section": "prefs"}), say("ok"),
		call(toolForget, m{"match": "健身"}),
		step(func(_ context.Context, in []*schema.Message) (*schema.Message, error) {
			contains(t, "forget result", lastToolText(in), "forgot 1")
			return call(toolRemember, m{"note": "健身归「生活」类，不是工作", "section": "prefs"})(nil, in)
		}),
		step(func(_ context.Context, in []*schema.Message) (*schema.Message, error) {
			contains(t, "remember result", lastToolText(in), "健身归「生活」类")
			return schema.AssistantMessage("好的", nil), nil
		}),
	)
	r.ask("健身归工作")
	res := r.ask("不对，健身归生活")
	if got := mem.Text(); !strings.Contains(got, "健身归「生活」类，不是工作") || strings.Contains(got, "「工作」类\n") {
		t.Fatalf("file:\n%s", got)
	}
	if len(res.Receipts) != 2 || res.Receipts[0].Detail.Kind != DetailForgot || res.Receipts[1].Detail.Kind != DetailRemembered {
		t.Fatalf("receipts %+v", res.Receipts)
	}
	if res.Receipts[1].Title != "健身归「生活」类，不是工作" || res.Receipts[0].Title != "健身" {
		t.Errorf("titles %+v", res.Receipts)
	}
	for i := len(res.Receipts) - 1; i >= 0; i-- { // undo both, newest first
		if err := r.agent.Undo(context.Background(), res.Receipts[i].ID); err != nil {
			t.Fatal(err)
		}
	}
	contains(t, "file after undo", mem.Text(), "健身归「工作」类")
	if strings.Contains(mem.Text(), "生活") {
		t.Errorf("undo left the new note:\n%s", mem.Text())
	}
}

func TestRememberErrorsAndDuplicatesMakeNoReceipt(t *testing.T) {
	r, mem := memRig(t,
		call(toolRemember, m{"note": "a", "section": "prefs"}), say("ok"),
		call(toolRemember, m{"note": "A", "section": "prefs"}), say("ok"),
		call(toolRemember, m{"note": "", "section": "prefs"}), say("ok"),
		call(toolForget, m{"match": "zzz"}), say("ok"),
	)
	if n := len(r.ask("1").Receipts); n != 1 {
		t.Errorf("first note: %d receipts", n)
	}
	for _, text := range []string{"2", "3", "4"} {
		if n := len(r.ask(text).Receipts); n != 0 {
			t.Errorf("%s: %d receipts", text, n)
		}
	}
	if n := countNotes(splitLines(mem.Text())); n != 1 {
		t.Errorf("%d notes", n)
	}
}

func TestNotesSurviveAnIdleReset(t *testing.T) {
	r, _ := memRig(t,
		call(toolRemember, m{"note": "叫我 Cyan", "section": "about"}), say("ok"),
		say("hello"),
	)
	r.ask("叫我 Cyan")
	r.clock.advance(DefaultIdleReset + time.Minute)
	r.ask("hi")
	last := r.model.sent(len(r.model.calls) - 1)
	for _, msg := range last {
		if msg.Role == schema.Tool {
			t.Fatal("the old conversation was not reset")
		}
	}
	contains(t, "prompt after reset", systemOf(last), "- 叫我 Cyan")
}

func TestHandEditsAreSeenOnTheNextTurn(t *testing.T) {
	r, mem := memRig(t, say("ok"))
	if err := os.MkdirAll(strings.TrimSuffix(mem.Path(), "assistant-memory.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mem.Path(), []byte("# 助手记忆\n## 关于我\n- 手写\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.ask("hi")
	contains(t, "prompt", systemOf(r.model.sent(0)), "- 手写")
}
