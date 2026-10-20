package main

import (
	"slices"
	"testing"

	"github.com/egoist/mygo"
)

// menuRoles lists the roles of a menu, the item itself for one without a role.
func menuRoles(items []*mygo.MenuItem) []string {
	var out []string
	for _, it := range items {
		switch {
		case it.Role != "":
			out = append(out, string(it.Role))
		case it.Type == mygo.MenuItemSeparator:
			out = append(out, "-")
		default:
			out = append(out, it.ID)
		}
	}
	return out
}

func TestMenuItemsPutSettingsInTheAppMenuOnAMac(t *testing.T) {
	settings, check := &mygo.MenuItem{ID: "settings"}, &mygo.MenuItem{ID: "check"}
	items := menuItems("darwin", settings, check)
	want := []string{"appMenu", "fileMenu", "editMenu", "viewMenu", "windowMenu"}
	if got := menuRoles(items); !slices.Equal(got, want) {
		t.Fatalf("darwin menus = %v, want %v", got, want)
	}
	wantApp := []string{"about", "check", "-", "settings", "-", "services", "-", "hide", "hideOthers", "unhide", "-", "quit"}
	if got := menuRoles(items[0].Submenu); !slices.Equal(got, wantApp) {
		t.Errorf("app menu = %v, want %v", got, wantApp)
	}
	if items[1].Submenu != nil {
		t.Error("the File menu is mygo's own on a Mac")
	}
}

func TestMenuItemsWithoutAnUpdateCheckLeaveItOut(t *testing.T) {
	settings := &mygo.MenuItem{ID: "settings"}
	items := menuItems("darwin", settings, nil)
	wantApp := []string{"about", "-", "settings", "-", "services", "-", "hide", "hideOthers", "unhide", "-", "quit"}
	if got := menuRoles(items[0].Submenu); !slices.Equal(got, wantApp) {
		t.Errorf("app menu = %v, want %v", got, wantApp)
	}
}

func TestMenuItemsPutSettingsAndQuitInTheFileMenuElsewhere(t *testing.T) {
	for _, goos := range []string{"windows", "linux"} {
		t.Run(goos, func(t *testing.T) {
			settings, check := &mygo.MenuItem{ID: "settings"}, &mygo.MenuItem{ID: "check"}
			items := menuItems(goos, settings, check)
			want := []string{"fileMenu", "editMenu", "viewMenu", "windowMenu"}
			if got := menuRoles(items); !slices.Equal(got, want) {
				t.Fatalf("menus = %v, want %v", got, want)
			}
			if got, want := menuRoles(items[0].Submenu), []string{"settings", "check", "-", "quit"}; !slices.Equal(got, want) {
				t.Errorf("File menu = %v, want %v", got, want)
			}
			items = menuItems(goos, settings, nil)
			if got, want := menuRoles(items[0].Submenu), []string{"settings", "-", "quit"}; !slices.Equal(got, want) {
				t.Errorf("File menu without an update check = %v, want %v", got, want)
			}
		})
	}
}
