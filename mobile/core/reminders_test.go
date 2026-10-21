package core

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/domain"
)

// TestClaimDueRemindersGolden pins the reminders' JSON shape: an Inbox task has a null project.
func TestClaimDueRemindersGolden(t *testing.T) {
	project := "Home"
	raw, err := encodeReminders([]domain.Reminder{
		{TaskID: 7, Title: "Pay rent", ProjectName: &project},
		{TaskID: 9, Title: "Stretch"},
	})
	if err != nil {
		t.Fatalf("encodeReminders: %v", err)
	}
	checkGolden(t, "reminders.golden.json", raw)
}

// TestDueRemindersAreClaimedOnceAndNextIsTheEarliest is the reminder behavior: a due reminder comes
// back once, NextReminderAt names the earliest open reminder, and done or already claimed tasks are
// left out.
func TestDueRemindersAreClaimedOnceAndNextIsTheEarliest(t *testing.T) {
	phone, err := open(t.TempDir(), nil, func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = phone.Close() }()

	first := createPhoneTask(t, phone, "Pay rent")
	second := createPhoneTask(t, phone, "Stretch")
	done := createPhoneTask(t, phone, "Already done")
	for id, remind := range map[int64]string{
		first:  "2026-10-20T08:00:00Z",
		second: "2026-10-20T10:00:00Z",
		done:   "2026-10-20T07:00:00Z",
	} {
		if err := phone.UpdateTask(id, `{"remind_at":"`+remind+`"}`); err != nil {
			t.Fatalf("set reminder on %d: %v", id, err)
		}
	}
	if err := phone.UpdateTask(done, `{"status":"done"}`); err != nil {
		t.Fatalf("complete the third task: %v", err)
	}

	wantFirst := time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC).UnixMilli()
	if got, err := phone.NextReminderAt(); err != nil || got != wantFirst {
		t.Fatalf("NextReminderAt = %d, %v; want %d", got, err, wantFirst)
	}

	raw, err := phone.ClaimDueReminders()
	if err != nil {
		t.Fatalf("ClaimDueReminders: %v", err)
	}
	var claimed []reminderJSON
	if err := json.Unmarshal([]byte(raw), &claimed); err != nil {
		t.Fatalf("decode reminders: %v (%s)", err, raw)
	}
	if len(claimed) != 1 || claimed[0].TaskID != first || claimed[0].Title != "Pay rent" || claimed[0].Project != nil {
		t.Fatalf("claimed = %+v, want only the first Inbox task", claimed)
	}
	if raw, err := phone.ClaimDueReminders(); err != nil || raw != "[]" {
		t.Errorf("second ClaimDueReminders = %q, %v; want []", raw, err)
	}

	wantSecond := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC).UnixMilli()
	if got, err := phone.NextReminderAt(); err != nil || got != wantSecond {
		t.Errorf("NextReminderAt after claiming = %d, %v; want %d", got, err, wantSecond)
	}

	// A task without a reminder leaves the next one unchanged, and a cleared reminder disappears.
	if err := phone.UpdateTask(second, `{"remind_at":null}`); err != nil {
		t.Fatalf("clear the second reminder: %v", err)
	}
	if got, err := phone.NextReminderAt(); err != nil || got != 0 {
		t.Errorf("NextReminderAt with no reminders = %d, %v; want 0", got, err)
	}
}

// createPhoneTask makes an Inbox task through the phone API and returns its id.
func createPhoneTask(t *testing.T, phone *Core, title string) int64 {
	t.Helper()
	raw, err := phone.CreateTask(`{"title":"` + title + `"}`)
	if err != nil {
		t.Fatalf("CreateTask %q: %v", title, err)
	}
	var task data.Task
	if err := json.Unmarshal([]byte(raw), &task); err != nil {
		t.Fatalf("decode %q: %v", title, err)
	}
	return int64(task.ID)
}

// TestRemindPresetsAreTheDesktopMenuAtAFixedNow pins the ready-made reminders the sheet offers and
// what they mean. The Sunday and the Saturday after 9:30 cases matter: they decide whether This
// weekend shows at all.
func TestRemindPresetsAreTheDesktopMenuAtAFixedNow(t *testing.T) {
	at := func(day, hour, min int) time.Time {
		return time.Date(2026, 10, day, hour, min, 0, 0, time.Local)
	}
	// Oct 12 2026 is a Monday, so Oct 17 is a Saturday and Oct 18 a Sunday.
	for _, c := range []struct {
		name string
		now  time.Time
		want []remindPresetJSON
	}{
		{"sunday offers tonight and the monday", at(18, 10, 0), []remindPresetJSON{
			{"later", at(18, 18, 0).UnixMilli()},
			{"tomorrow", at(19, 9, 0).UnixMilli()},
		}},
		{"saturday after 9:30 has no weekend left", at(17, 12, 0), []remindPresetJSON{
			{"later", at(17, 18, 0).UnixMilli()},
			{"tomorrow", at(18, 9, 0).UnixMilli()},
			{"monday", at(19, 9, 0).UnixMilli()},
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			phone, err := open(t.TempDir(), nil, func() time.Time { return c.now })
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer func() { _ = phone.Close() }()
			raw, err := phone.RemindPresets()
			if err != nil {
				t.Fatalf("RemindPresets: %v", err)
			}
			var got []remindPresetJSON
			if err := json.Unmarshal([]byte(raw), &got); err != nil {
				t.Fatalf("decode presets: %v (%s)", err, raw)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("presets = %+v, want %+v", got, c.want)
			}
		})
	}
}
