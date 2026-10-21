package core

import "testing"

// A new database has no projects; Projects and Categories must still be JSON arrays, or the app's
// list decoders fail and a task page or the task menu stays empty.
func TestListsOfANewDatabaseAreArrays(t *testing.T) {
	phone, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	for name, read := range map[string]func() (string, error){
		"Projects": phone.Projects, "Categories": phone.Categories,
	} {
		got, err := read()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got == "null" || got == "" || got[0] != '[' {
			t.Errorf("%s = %s, want a JSON array", name, got)
		}
	}
}
