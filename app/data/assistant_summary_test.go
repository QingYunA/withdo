package data

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func say(text string) func(context.Context) (*schema.Message, error) {
	return func(context.Context) (*schema.Message, error) { return schema.AssistantMessage(text, nil), nil }
}

func summaryRequest() SummaryRequest {
	return SummaryRequest{
		Label: "Friday 2026-10-16", Language: "English", Open: SummaryOpen{InProgress: 2},
		Tasks: []SummaryRequestTask{{ID: 3, Title: "Pay rent", Who: You}, {ID: 4, Title: "Fix the leak", Who: "Codex", PR: 9, Merged: true}},
	}
}

func TestAgentAssistantSummarizesInOneCallAndNamesTheModel(t *testing.T) {
	a, _ := newAssistant(t, say(`{"paragraph":"You paid the rent.","kinds":{"3":"coin","4":"jar"}}`))
	got, err := a.Summarize(context.Background(), summaryRequest())
	if err != nil {
		t.Fatal(err)
	}
	if got.Paragraph != "You paid the rent." || got.Kinds[3] != "coin" || got.Kinds[4] != "jar" {
		t.Errorf("%+v", got)
	}
}

func TestAgentAssistantSummaryFailureIsInPlainWords(t *testing.T) {
	fail := func(context.Context) (*schema.Message, error) { return nil, errors.New("401 invalid x-api-key") }
	a, _ := newAssistant(t, fail)
	_, err := a.Summarize(context.Background(), summaryRequest())
	if ProblemOf(err) != ProblemKeyRejected {
		t.Errorf("err = %v (%v)", err, ProblemOf(err))
	}
}

func TestAgentAssistantSummaryCancelledIsCancelled(t *testing.T) {
	a, _ := newAssistant(t, func(ctx context.Context) (*schema.Message, error) { <-ctx.Done(); return nil, ctx.Err() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Summarize(ctx, summaryRequest()); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
}

func TestFakeAssistantAnswersSummariesFromItsScript(t *testing.T) {
	f := &FakeAssistant{Ready: true}
	got, err := f.Summarize(context.Background(), summaryRequest())
	if err != nil || got.Paragraph == "" || len(f.Summaries()) != 1 {
		t.Errorf("%+v %v %d", got, err, len(f.Summaries()))
	}
}
