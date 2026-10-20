package main

import (
	"slices"
	"testing"

	"github.com/egoist/mygo"
)

func TestTrayItemsOpenSettingsAndQuit(t *testing.T) {
	var opened, settings int
	items := trayItems(func() { opened++ }, func() { settings++ })
	if got, want := menuRoles(items), []string{"open", "settings", "-", "quit"}; !slices.Equal(got, want) {
		t.Fatalf("tray menu = %v, want %v", got, want)
	}
	items[0].Click(items[0], nil)
	items[1].Click(items[1], nil)
	if opened != 1 || settings != 1 {
		t.Errorf("Open ran %d times and Settings %d, want once each", opened, settings)
	}
	if items[3].Role != mygo.RoleQuit {
		t.Errorf("last item = %+v, want Quit", items[3])
	}
}
