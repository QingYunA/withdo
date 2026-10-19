package syncproto_test

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

var (
	updateTrigger = regexp.MustCompile(`^sync_(\w+)_update AFTER UPDATE`)
	triggerColumn = regexp.MustCompile(`SELECT '([a-z_]+)' AS name, \d+ AS n WHERE OLD\.`)
)

// TestS2a7ColumnsMatchMigration014 is acceptance criterion 7's second half: the server's column
// lists and the SQLite triggers that fill the outbox name the same columns, in the same order, so
// the two cannot drift.
func TestS2a7ColumnsMatchMigration014(t *testing.T) {
	raw, err := os.ReadFile("../store/migrations/014_sync.sql")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, block := range strings.Split(string(raw), "CREATE TRIGGER ") {
		m := updateTrigger.FindStringSubmatch(block)
		if m == nil {
			continue
		}
		entity := m[1]
		for _, c := range triggerColumn.FindAllStringSubmatch(block, -1) {
			got[entity] = append(got[entity], c[1])
		}
	}
	if len(got) != len(syncproto.Columns) {
		t.Fatalf("migration 014 has update triggers for %d entities, syncproto has %d", len(got), len(syncproto.Columns))
	}
	for entity, want := range syncproto.Columns {
		if !reflect.DeepEqual(got[entity], want) {
			t.Errorf("%s columns = %v, migration trigger has %v", entity, want, got[entity])
		}
	}
}
