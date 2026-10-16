package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func handed(id int, at time.Time) Item { return Item{Task: Task{ID: id}, Since: at} }

func TestSeenHasAnItemUntilItIsHandedOverAgain(t *testing.T) {
	at := fixedNow()
	s := Seen{1: at}
	if !s.Has(handed(1, at)) || !s.Has(handed(1, at.Add(-time.Hour))) {
		t.Error("an item handed over at or before the seen time is seen")
	}
	if s.Has(handed(1, at.Add(time.Second))) {
		t.Error("an item handed over again, later, is unseen again")
	}
	if s.Has(handed(2, at)) {
		t.Error("an item never seen is unseen")
	}
	got := Unseen([]Item{handed(1, at), handed(2, at), handed(3, at)}, s)
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 3 {
		t.Errorf("unseen %v, want 2 and 3 in order", got)
	}
}

func TestMarkSeenReturnsACopyAndForgetsGoneTasks(t *testing.T) {
	at := fixedNow()
	s := Seen{1: at, 9: at}
	next, changed := s.MarkSeen([]Item{handed(1, at), handed(2, at)}, func(id int) bool { return id != 9 })
	if !changed || len(next) != 2 || !next[1].Equal(at) || !next[2].Equal(at) {
		t.Errorf("next %v changed %v", next, changed)
	}
	if len(s) != 2 || s[9].IsZero() {
		t.Errorf("the original was changed: %v", s)
	}
	if _, changed := next.MarkSeen([]Item{handed(2, at)}, nil); changed {
		t.Error("marking what is seen reports a change")
	}
}

func TestFileSeenRoundTripsAndToleratesAMissingFile(t *testing.T) {
	store := FileSeen{Path: filepath.Join(t.TempDir(), "sub", "seen.json")}
	if got, err := store.LoadSeen(); err != nil || len(got) != 0 {
		t.Fatalf("a missing file: %v, %v", got, err)
	}
	at := fixedNow()
	if err := store.SaveSeen(Seen{4: at, 7: at.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadSeen()
	if err != nil || len(got) != 2 || !got[4].Equal(at) || !got[7].Equal(at.Add(time.Minute)) {
		t.Errorf("round trip: %v, %v", got, err)
	}
}

func TestFileSeenReportsACorruptFileAndKeepsGoing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seen.json")
	if err := os.WriteFile(path, []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := FileSeen{Path: path}.LoadSeen()
	if err == nil || got == nil {
		t.Errorf("corrupt file: %v, %v: want an empty record and an error", got, err)
	}
}

func TestMemorySeenKeepsACopy(t *testing.T) {
	var m MemorySeen
	s := Seen{1: fixedNow()}
	_ = m.SaveSeen(s)
	s[2] = fixedNow()
	got, _ := m.LoadSeen()
	if len(got) != 1 {
		t.Errorf("stored %v", got)
	}
}

func TestWithoutDismissedHidesARowUntilItIsHandedOverAgain(t *testing.T) {
	at := fixedNow()
	m := Mine{FromAgents: []Item{handed(1, at), handed(2, at)}, Today: []Item{handed(3, at)}}
	d := Seen{1: at, 3: at}
	got := m.WithoutDismissed(d)
	if len(got.FromAgents) != 1 || got.FromAgents[0].ID != 2 || got.Count() != 2 {
		t.Errorf("from agents %v, count %d: row 1 is dismissed", got.FromAgents, got.Count())
	}
	if len(got.Today) != 1 {
		t.Error("only what agents handed over can be dismissed")
	}
	if len(m.FromAgents) != 2 {
		t.Error("the original was changed")
	}
	again := Mine{FromAgents: []Item{handed(1, at.Add(time.Minute))}}.WithoutDismissed(d)
	if len(again.FromAgents) != 1 {
		t.Error("an agent that reports again brings the row back")
	}
}
