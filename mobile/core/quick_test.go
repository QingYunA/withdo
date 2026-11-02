package core

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
	"unicode/utf16"
)

func parseQuick(t *testing.T, phone *Core, text string) []quickDraftJSON {
	t.Helper()
	raw, err := phone.ParseQuick(text)
	if err != nil {
		t.Fatal(err)
	}
	var got quickJSON
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("decode ParseQuick: %v", err)
	}
	return got.Drafts
}

// utf16Slice cuts s the way Kotlin's substring does: by UTF-16 units.
func utf16Slice(s string, r [2]int) string {
	return string(utf16.Decode(utf16.Encode([]rune(s))[r[0]:r[1]]))
}

// TestParseQuickReadsTheWordsAndSaysWhere pins ParseQuick: demoNow is Tuesday 2026-10-20 09:00 UTC.
func TestParseQuickReadsTheWordsAndSaysWhere(t *testing.T) {
	phone := demoPhone(t)

	t.Run("a reminder in Chinese", func(t *testing.T) {
		text := "提醒我明天9点交房租"
		drafts := parseQuick(t, phone, text)
		if len(drafts) != 1 {
			t.Fatalf("drafts = %+v", drafts)
		}
		d := drafts[0]
		if d.Title != "交房租" || d.RemindAt != "2026-10-21T09:00:00Z" || d.DueAt != "" {
			t.Errorf("draft = %+v", d)
		}
		var read []string
		for _, r := range d.Ranges {
			read = append(read, utf16Slice(text, r))
		}
		if !reflect.DeepEqual(read, []string{"提醒我明天9点"}) {
			t.Errorf("ranges %v read %q", d.Ranges, read)
		}
	})

	t.Run("a deadline in English with a category and a star", func(t *testing.T) {
		text := "write the report by Friday #life !"
		d := parseQuick(t, phone, text)[0]
		if d.Title != "write the report" || d.DueAt != "2026-10-23T18:00:00Z" || d.RemindAt != "" || !d.Important {
			t.Errorf("draft = %+v", d)
		}
		if d.CategoryID != 2 || d.Category != "Life" {
			t.Errorf("category = %d %q, want Life (2)", d.CategoryID, d.Category)
		}
		var read []string
		for _, r := range d.Ranges {
			read = append(read, utf16Slice(text, r))
		}
		if !reflect.DeepEqual(read, []string{"by Friday", "#life", "!"}) {
			t.Errorf("read %q", read)
		}
	})

	t.Run("the shape the app decodes", func(t *testing.T) {
		raw, err := phone.ParseQuick("提醒我明天9点交房租 #生活 !")
		if err != nil {
			t.Fatal(err)
		}
		checkGolden(t, "quick.golden.json", raw)
	})

	t.Run("work and life answer to their Chinese names", func(t *testing.T) {
		d := parseQuick(t, phone, "买菜 #生活")[0]
		if d.CategoryID != 2 || d.Title != "买菜" {
			t.Errorf("draft = %+v", d)
		}
	})

	t.Run("offsets count UTF-16 units, not bytes or code points", func(t *testing.T) {
		text := "😀😀 明天交房租" // each emoji is two UTF-16 units
		d := parseQuick(t, phone, text)[0]
		if len(d.Ranges) != 1 || utf16Slice(text, d.Ranges[0]) != "明天" || d.Ranges[0] != [2]int{5, 7} {
			t.Errorf("ranges = %v", d.Ranges)
		}
	})

	t.Run("plain text reads nothing", func(t *testing.T) {
		d := parseQuick(t, phone, "buy milk")[0]
		if d.Title != "buy milk" || d.RemindAt != "" || d.DueAt != "" || len(d.Ranges) != 0 || d.CategoryID != 0 {
			t.Errorf("draft = %+v", d)
		}
	})

	t.Run("blank text and separators", func(t *testing.T) {
		if got := parseQuick(t, phone, "  "); len(got) != 0 {
			t.Errorf("blank = %+v", got)
		}
		text := "买牛奶；后天体检"
		got := parseQuick(t, phone, text)
		if len(got) != 2 || got[1].Title != "体检" || utf16Slice(text, got[1].Ranges[0]) != "后天" {
			t.Errorf("two tasks = %+v", got)
		}
	})
}

// TestCreateTaskStoresWhatWasRead: remind_at round-trips through CreateTask, and the two milestone
// and notes fields of UpdateTask round-trip too.
func TestCreateTaskStoresWhatWasRead(t *testing.T) {
	phone := demoPhone(t)
	d := parseQuick(t, phone, "提醒我明天9点交房租 #生活 !")[0]
	input, _ := json.Marshal(map[string]any{
		"title": d.Title, "remind_at": d.RemindAt, "category_id": d.CategoryID, "priority": "P1",
	})
	raw, err := phone.CreateTask(string(input))
	if err != nil {
		t.Fatal(err)
	}
	var made struct {
		ID         int
		Title      string
		RemindAt   *time.Time
		DueAt      *time.Time
		CategoryID int
		Priority   string
	}
	if err := json.Unmarshal([]byte(raw), &made); err != nil {
		t.Fatal(err)
	}
	if made.Title != "交房租" || made.RemindAt == nil || !made.RemindAt.Equal(time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)) ||
		made.DueAt != nil || made.CategoryID != 2 || made.Priority != "P1" {
		t.Errorf("created %+v", made)
	}
	if _, err := phone.CreateTask(`{"title":"x","remind_at":"someday"}`); err == nil {
		t.Error("a remind_at that is not a date was accepted")
	}

	// Task 1 is in project 1 and milestone 1: every field a task page edits round-trips.
	if err := phone.UpdateTask(1, `{"notes":"n","due_at":"2026-10-30T10:00:00Z","remind_at":"2026-10-29T10:00:00Z","category":2,"milestone_id":null}`); err != nil {
		t.Fatal(err)
	}
	task := mustTask(t, phone, 1)
	if task.MilestoneID != nil || task.Notes != "n" || task.CategoryID != 2 || task.DueAt == nil || task.RemindAt == nil || task.ProjectID == nil || *task.ProjectID != 1 {
		t.Errorf("after the patch: %+v", task)
	}
	if err := phone.UpdateTask(1, `{"milestone_id":1}`); err != nil {
		t.Fatal(err)
	}
	if task := mustTask(t, phone, 1); task.MilestoneID == nil || *task.MilestoneID != 1 {
		t.Errorf("milestone after setting 1: %+v", task.MilestoneID)
	}
	if err := phone.UpdateTask(1, `{"milestone_id":0}`); err != nil {
		t.Fatal(err)
	}
	if task := mustTask(t, phone, 1); task.MilestoneID != nil {
		t.Errorf("milestone after 0: %+v", task.MilestoneID)
	}
	// Moving to another project takes the milestone along only when the patch names one.
	if err := phone.UpdateTask(1, `{"milestone_id":1}`); err != nil {
		t.Fatal(err)
	}
	if err := phone.UpdateTask(1, `{"project_id":2}`); err != nil {
		t.Fatal(err)
	}
	if task := mustTask(t, phone, 1); task.MilestoneID != nil || *task.ProjectID != 2 {
		t.Errorf("after moving to project 2: %+v", task)
	}
}
