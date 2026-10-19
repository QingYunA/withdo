package agent

import (
	"context"
	"fmt"
	"strings"
)

// Tool names of the assistant's memory.
const (
	toolRemember = "remember"
	toolForget   = "forget"
)

// memoryRules tells the model when to use the memory. It is part of the prompt only when there is one.
const memoryRules = `
Memory (remember and forget keep short notes about the user between conversations):
- When the user corrects you ("不对", "不是…是…", "别…", "以后/下次…", always, never) or states a lasting preference, first apply the correction to the task itself, then call remember with a short rule that generalizes (in the user's language, no task ids, no one-off dates), section "prefs".
- When the user tells you about themselves (their name or how to address them, language, working hours, habits), call remember with section "about".
- When a new note contradicts an old one, call forget for the old one first, then remember the new one.
- Do not remember one-off requests, or anything you can read from the task data.
`

// userNotes is the block of the prompt with the user's notes; empty when there are none.
func userNotes(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return "\nUser notes (from the user's corrections and preferences; obey them over the default rules above):\n" + text + "\n"
}

type rememberArgs struct {
	Note    string `json:"note" jsonschema:"description=One short rule or fact in the user's language. No task ids or one-off dates. At most 200 characters."`
	Section string `json:"section" jsonschema:"enum=about,enum=prefs,description=about is for the user (how to address them, language, schedule and habits); prefs is for corrections and preferences."`
}

func (tb *toolbox) remember(ctx context.Context, a rememberArgs) (reply, error) {
	e, err := tb.memory.Add(a.Section, a.Note)
	if err != nil {
		return fail(ctx, err)
	}
	if e.Changed == 0 {
		return reply{OK: true, Note: "already remembered", Memory: e.Text}, nil
	}
	note, _ := cleanNote(a.Note)
	tb.addReceipt(Receipt{Title: note, Detail: Detail{Kind: DetailRemembered}}, tb.undoMemory(e))
	return reply{OK: true, Memory: e.Text}, nil
}

type forgetArgs struct {
	Match string `json:"match" jsonschema:"description=Text that occurs in the notes to delete. Every note containing it is removed."`
}

func (tb *toolbox) forget(ctx context.Context, a forgetArgs) (reply, error) {
	e, err := tb.memory.Forget(a.Match)
	if err != nil {
		return fail(ctx, err)
	}
	if e.Changed == 0 {
		return reply{OK: true, Note: "no note contains that text", Memory: e.Text}, nil
	}
	tb.addReceipt(Receipt{Title: strings.TrimSpace(a.Match), Detail: Detail{Kind: DetailForgot}}, tb.undoMemory(e))
	return reply{OK: true, Note: fmt.Sprintf("forgot %d note(s)", e.Changed), Memory: e.Text}, nil
}

// undoMemory puts the file back as it was before the edit.
func (tb *toolbox) undoMemory(e MemoryEdit) func(context.Context) error {
	return func(context.Context) error { return tb.memory.Restore(e) }
}
