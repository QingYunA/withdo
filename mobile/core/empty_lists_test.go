package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"
)

// notCalled lists the argument-free (string, error) methods the test below must not call. Every
// other one is called on a new database, so a method added later is checked without touching this
// file; add it here only if calling it has a side effect or needs a network, and say why.
var notCalled = map[string]string{
	"Devices":         "asks the sync server",
	"GoogleStart":     "opens a Google sign-in",
	"GoogleLinkStart": "opens a Google link flow",
}

// A new database has no rows; every JSON a Core method returns must still be free of null, or the
// app's decoders fail (a nil slice is null, and the Kotlin list decoder wants a '[').
func TestJSONOfANewDatabaseHasNoNull(t *testing.T) {
	phone, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = phone.Close() })

	stringErr := reflect.TypeOf((func() (string, error))(nil))
	value := reflect.ValueOf(phone)
	var called []string
	for i := 0; i < value.NumMethod(); i++ {
		name := value.Type().Method(i).Name
		method := value.Method(i)
		if method.Type() != stringErr {
			continue
		}
		if _, skip := notCalled[name]; skip {
			continue
		}
		called = append(called, name)
		out := method.Call(nil)
		if err, _ := out[1].Interface().(error); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		raw := out[0].String()
		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			t.Errorf("%s = %q, not JSON: %v", name, raw, err)
			continue
		}
		if path := findNull(decoded, name); path != "" {
			t.Errorf("%s has a null at %s: %s", name, path, raw)
		}
	}
	for name := range notCalled {
		if _, ok := value.Type().MethodByName(name); !ok {
			t.Errorf("notCalled lists %s, which Core no longer has", name)
		}
	}
	sort.Strings(called)
	t.Logf("checked %v", called)
}

// findNull returns the path of the first JSON null in v, or "".
func findNull(v any, path string) string {
	switch v := v.(type) {
	case nil:
		return path
	case []any:
		for i, item := range v {
			if found := findNull(item, fmt.Sprintf("%s[%d]", path, i)); found != "" {
				return found
			}
		}
	case map[string]any:
		for key, item := range v {
			if found := findNull(item, path+"."+key); found != "" {
				return found
			}
		}
	}
	return ""
}
