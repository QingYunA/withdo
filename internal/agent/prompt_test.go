package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestPromptHasClockZoneProjectsAndLanguageRule(t *testing.T) {
	loc := loadZone(t, "Asia/Shanghai")
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, loc)
	p := BuildPrompt(now, []domain.Project{{ID: 3, Name: "adhd-todo"}, {ID: 4, Name: "picpocket"}}, Scope{}, nil)
	for _, want := range []string{
		"Wednesday, 2026-10-14 14:30",
		"Asia/Shanghai",
		"UTC+08:00",
		"2026-10-14T14:30:00+08:00",
		"Thu 2026-10-15",
		"stays +08:00",
		"- 3, adhd-todo",
		"- 4, picpocket",
		"Reply in the language the user wrote in",
		"ask_user",
		"never invent task ids",
	} {
		if !strings.Contains(strings.ToLower(p), strings.ToLower(want)) {
			t.Errorf("prompt lacks %q:\n%s", want, p)
		}
	}
}

// The user writes Chinese as often as English, so the inference rules carry the Chinese words too,
// and one message may hold several tasks or report one done.
func TestPromptInfersFromChineseAndSplitsLists(t *testing.T) {
	p := BuildPrompt(time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC), nil, Scope{}, nil)
	for _, want := range []string{"紧急", "不急", "线上挂了", "明早", "下班前", "周末", "separate tasks", "complete_task", "Titles are short"} {
		contains(t, "prompt", p, want)
	}
}

func TestPromptWithoutProjects(t *testing.T) {
	p := BuildPrompt(time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC), nil, Scope{}, nil)
	contains(t, "prompt", p, "none yet")
	contains(t, "prompt", p, "UTC+00:00")
}

func TestPromptSkipsArchivedProjects(t *testing.T) {
	arch := "2026-01-01T00:00:00.000Z"
	p := BuildPrompt(time.Now(), []domain.Project{{ID: 1, Name: "live"}, {ID: 2, Name: "gone", ArchivedAt: &arch}}, Scope{}, nil)
	contains(t, "prompt", p, "live")
	if strings.Contains(p, "gone") {
		t.Error("archived project listed")
	}
}

func TestPromptSpellsOutDaylightSavingChange(t *testing.T) {
	loc := loadZone(t, "America/New_York")
	now := time.Date(2026, 3, 7, 20, 0, 0, 0, loc) // the evening before the clocks go forward
	p := BuildPrompt(now, nil, Scope{}, nil)
	contains(t, "prompt", p, "-05:00 today and changes")
	contains(t, "prompt", p, "-04:00 from 2026-03-08")
	contains(t, "prompt", p, "Sun 2026-03-08")
	if strings.Contains(p, "stays") {
		t.Error("claims the offset is fixed")
	}
}

func TestStoredTimeIsInTheZoneOfNow(t *testing.T) {
	loc := loadZone(t, "Asia/Shanghai")
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, loc)
	got := storedTime("2026-10-15T01:00:00.000Z", now)
	if got.Location() != loc || got.Hour() != 9 {
		t.Errorf("storedTime = %v, want 9:00 in Shanghai", got)
	}
	if !storedTime("garbage", now).IsZero() {
		t.Error("a malformed timestamp is not the zero time")
	}
}

func TestParseWhen(t *testing.T) {
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC)
	good := map[string]string{
		"2026-10-15T09:00:00+08:00": "2026-10-15T01:00:00.000Z",
		"2026-10-15T09:00:00Z":      "2026-10-15T09:00:00.000Z",
		" 2026-10-14T14:30:30Z ":    "2026-10-14T14:30:30.000Z", // inside the grace window
	}
	for in, want := range good {
		got, err := parseWhen("remind_at", in, now, true)
		if err != nil || got != want {
			t.Errorf("parseWhen(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"2026-10-15T09:00:00", "2026-10-15", "tomorrow 9am", "2026-10-15 09:00", ""} {
		if _, err := parseWhen("remind_at", in, now, true); err == nil {
			t.Errorf("parseWhen(%q) accepted a time without a zone", in)
		}
	}
	if _, err := parseWhen("remind_at", "2026-10-14T14:00:00Z", now, true); err == nil {
		t.Error("accepted a reminder in the past")
	}
	if _, err := parseWhen("due_at", "2026-10-14T14:00:00Z", now, false); err != nil {
		t.Errorf("a past due date is allowed: %v", err)
	}
	if got := localRFC3339(nil, time.UTC); got != "" {
		t.Errorf("localRFC3339(nil) = %q", got)
	}
	bad := "oops"
	eq(t, "unparsable stored time", localRFC3339(&bad, time.UTC), "oops")
}
