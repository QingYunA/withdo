package core

import (
	"reflect"
	"sort"
	"testing"
)

// TestTheExportedAPIIsBindable is the gomobile rule as a test: every exported function and method
// takes and returns only types gomobile can bind, and every exported interface's methods do too.
func TestTheExportedAPIIsBindable(t *testing.T) {
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	seen := map[reflect.Type]bool{}
	var bindable func(reflect.Type) bool
	bindableStruct := func(typ reflect.Type) bool {
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.PkgPath != "" { // unexported
				continue
			}
			if !bindable(f.Type) {
				t.Logf("field %s.%s has unbindable type %s", typ, f.Name, f.Type)
				return false
			}
		}
		return true
	}
	bindableInterface := func(typ reflect.Type) bool {
		for i := 0; i < typ.NumMethod(); i++ {
			if !bindableSignature(typ.Method(i).Type, bindable) {
				t.Logf("interface %s method %s is unbindable", typ, typ.Method(i).Name)
				return false
			}
		}
		return true
	}
	bindable = func(t reflect.Type) bool {
		if seen[t] {
			return true
		}
		seen[t] = true
		switch t.Kind() {
		case reflect.String, reflect.Bool, reflect.Int64, reflect.Float64:
			return true
		case reflect.Slice:
			return t.Elem().Kind() == reflect.Uint8 // []byte
		case reflect.Ptr:
			return t.Elem().Kind() == reflect.Struct && bindableStruct(t.Elem())
		case reflect.Struct:
			return bindableStruct(t)
		case reflect.Interface:
			if t == errorType {
				return true
			}
			return bindableInterface(t)
		}
		return false
	}

	if typ := reflect.TypeOf(Open); !bindableSignature(typ, bindable) {
		t.Errorf("Open is not bindable: %s", typ)
	}
	if typ := reflect.TypeOf(SeedDemo); !bindableSignature(typ, bindable) {
		t.Errorf("SeedDemo is not bindable: %s", typ)
	}
	coreType := reflect.TypeOf((*Core)(nil))
	var methods []string
	for i := 0; i < coreType.NumMethod(); i++ {
		method := coreType.Method(i)
		methods = append(methods, method.Name)
		if !bindableSignature(method.Type, bindable) {
			t.Errorf("%s is not bindable: %s", method.Name, method.Type)
		}
	}
	sort.Strings(methods)
	want := []string{
		"Accept", "Account", "Agents", "Categories", "ClaimDueReminders", "Close", "CodeState",
		"Complete", "CreateTask", "DeleteTask", "Devices", "DoToday", "GoogleAvailable", "GoogleCancel", "GoogleFinish", "GoogleLinkAvailable", "GoogleLinkCancel",
		"GoogleLinkFinish", "GoogleLinkStart", "GoogleLinked", "GoogleStart", "LinkMerge", "LinkReplace", "Mine",
		"MoveToAccount", "NextReminderAt",
		"Project", "Projects", "RemindPresets", "Reply", "RequestCode",
		"RequestPasswordReset", "ResendCode", "RestoreTask", "Search", "SetListener",
		"SignIn", "SignInWithCode", "SignOut", "StartSync", "StopSync", "SyncNow", "SyncSetServer",
		"SyncStatus", "Task", "UpdateTask",
	}
	if !reflect.DeepEqual(methods, want) {
		t.Errorf("Core methods = %v\nwant %v", methods, want)
	}
	for _, typ := range []reflect.Type{
		reflect.TypeOf((*Secrets)(nil)).Elem(),
		reflect.TypeOf((*Listener)(nil)).Elem(),
	} {
		if !bindable(typ) {
			t.Errorf("%s is not bindable", typ)
		}
	}
}

// bindableSignature checks every argument and result of a function or method type.
func bindableSignature(t reflect.Type, bindable func(reflect.Type) bool) bool {
	for i := 0; i < t.NumIn(); i++ {
		if !bindable(t.In(i)) {
			return false
		}
	}
	for i := 0; i < t.NumOut(); i++ {
		if !bindable(t.Out(i)) {
			return false
		}
	}
	return true
}
