package core

import (
	"testing"
	"time"
)

func TestSetZoneMakesThePhonesZoneLocal(t *testing.T) {
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })

	SetZone("Asia/Shanghai", 8*3600)
	if got := time.Now().Location().String(); got != "Asia/Shanghai" {
		t.Fatalf("time.Now is in %s, want Asia/Shanghai", got)
	}
	if _, off := time.Now().Zone(); off != 8*3600 {
		t.Fatalf("offset %d, want %d", off, 8*3600)
	}
}

func TestSetZoneFallsBackToTheOffset(t *testing.T) {
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })

	SetZone("Nowhere/Unknown", -5*3600)
	if _, off := time.Now().Zone(); off != -5*3600 {
		t.Fatalf("offset %d, want %d", off, -5*3600)
	}
}

func TestTomorrowAtNineIsReadInThePhonesZone(t *testing.T) {
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })
	time.Local = time.UTC // what Go on Android starts with
	SetZone("Asia/Shanghai", 8*3600)
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Skip("no zone database")
	}

	// The real clock, as Open has it: time.Now in time.Local.
	phone, err := open(t.TempDir(), nil, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	drafts := parseQuick(t, phone, "提醒我明天9点交房租")
	if len(drafts) != 1 {
		t.Fatalf("drafts = %+v", drafts)
	}
	at, err := time.Parse(time.RFC3339, drafts[0].RemindAt)
	if err != nil {
		t.Fatalf("remind_at %q: %v", drafts[0].RemindAt, err)
	}
	if h := at.In(shanghai).Hour(); h != 9 {
		t.Fatalf("reminder at %s, want 9:00 in Asia/Shanghai", at)
	}
}
