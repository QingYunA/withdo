package views

import (
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
)

// The project menu and the milestone menus are drawn in the window (PopoverMenu,
// PopoverMenuButton): the system shows none, so tt.Menu and tt.ChooseMenuItem do not see them.

// drawnMenu does open, which opens a drawn menu, and returns the menu's items, without separators.
// The menu is built after the window, so its items are the texts it adds at the end; each shows its
// label twice, as the item's name and as its text.
func drawnMenu(tt *ui.Tester, open func()) []string {
	n := len(tt.Texts())
	open()
	tt.Frame()
	return slices.Compact(slices.Clone(tt.Texts()[n:]))
}

// chooseItem chooses the item of the drawn menu that shows at path, the labels of the items to go
// through, as the pointer does: it rests on each submenu's item, which opens the submenu, and
// clicks the last.
func chooseItem(tt *ui.Tester, path ...string) error {
	if len(path) == 0 {
		return fmt.Errorf("no item to choose")
	}
	for i, label := range path {
		r, ok := tt.Find(label)
		if !ok {
			return fmt.Errorf("the menu has no item %q", label)
		}
		x, y := r.X+r.W/2, r.Y+r.H/2
		if i == len(path)-1 {
			tt.ClickAt(x, y)
			break
		}
		tt.Move(x, y)
		tt.Frame()
	}
	tt.Frame()
	return nil
}
