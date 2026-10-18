package agent

import (
	"context"
	"strings"
	"testing"
)

func summaryInput() SummaryInput {
	return SummaryInput{
		Label: "Friday 2026-10-17", Language: "English", InProgress: 3, Blocked: 1,
		Tasks: []SummaryTask{
			{ID: 12, Title: "Native UI with mygo", Project: "adhd-todo", Who: "Claude Code", PR: 12, Merged: true, Said: "Board, lists and inspector are in"},
			{ID: 14, Title: "Pay rent", Who: "you"},
		},
	}
}

func TestSummarizeAsksOnceWithTheTasksAndReadsTheJSON(t *testing.T) {
	r := newRig(t, "UTC", oct14, say(`{"paragraph":"You paid the rent.","kinds":{"12":"gem","14":"coin"}}`))
	got, err := r.agent.Summarize(context.Background(), summaryInput())
	if err != nil {
		t.Fatal(err)
	}
	if got.Paragraph != "You paid the rent." || got.Kinds[12] != "gem" || got.Kinds[14] != "coin" {
		t.Errorf("got %+v", got)
	}
	sent := r.model.sent(0)
	if len(sent) != 2 {
		t.Fatalf("sent %d messages, want the prompt and the tasks", len(sent))
	}
	contains(t, "prompt", sent[0].Content, "Write the paragraph in English")
	contains(t, "tasks", sent[1].Content, "#12 | Native UI with mygo | by Claude Code | project adhd-todo | PR #12 merged | last report: Board, lists and inspector are in")
	contains(t, "tasks", sent[1].Content, "#14 | Pay rent | by you")
	contains(t, "open", sent[1].Content, "Still open: 3 in progress, 0 in review, 1 blocked.")
}

func TestTheSummaryPromptForbidsInventingCheeringAndPreaching(t *testing.T) {
	p := summaryPrompt(summaryInput())
	for _, want := range []string{"Never invent", "Do not cheer, praise, preach or advise", "Use only the tasks you are given"} {
		contains(t, "prompt", p, want)
	}
	for _, kind := range []string{"gem", "jar", "gear", "book", "flask", "mail", "coin", "scroll", "crate"} {
		contains(t, "prompt", p, kind+" (")
	}
}

func TestSummarizeReadsJSONInACodeFenceAndDropsWhatItCannotUse(t *testing.T) {
	reply := "Here you go:\n```json\n{\"paragraph\": \" Two done. \", \"kinds\": {\"12\": \"Gem\", \"14\": \"dragon\", \"99\": \"jar\", \"#14\": \"coin\"}}\n```"
	r := newRig(t, "UTC", oct14, say(reply))
	got, err := r.agent.Summarize(context.Background(), summaryInput())
	if err != nil {
		t.Fatal(err)
	}
	if got.Paragraph != "Two done." {
		t.Errorf("paragraph %q", got.Paragraph)
	}
	if len(got.Kinds) != 2 || got.Kinds[12] != "gem" || got.Kinds[14] != "coin" {
		t.Errorf("kinds %v: 12 is a gem, 14 a coin, 99 was not given and dragon is no good", got.Kinds)
	}
}

func TestSummarizeFailsWithoutAParagraphOrJSON(t *testing.T) {
	for _, reply := range []string{"I could not.", `{"paragraph": "", "kinds": {}}`, `{"paragraph": 3}`} {
		r := newRig(t, "UTC", oct14, say(reply))
		if _, err := r.agent.Summarize(context.Background(), summaryInput()); err == nil {
			t.Errorf("%q was accepted", reply)
		}
	}
}

func TestSummarizeNeedsTasksAndPassesTheModelsFailureOn(t *testing.T) {
	r := newRig(t, "UTC", oct14, fail401)
	if _, err := r.agent.Summarize(context.Background(), SummaryInput{}); err == nil {
		t.Error("an empty period was summarized")
	}
	if _, err := r.agent.Summarize(context.Background(), summaryInput()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want the provider's 401", err)
	}
}

func TestALongReportIsCutToOneLine(t *testing.T) {
	got := oneLine("a\n\nb  " + strings.Repeat("x", 300))
	if strings.Contains(got, "\n") || len([]rune(got)) > 241 {
		t.Errorf("%q", got)
	}
}
