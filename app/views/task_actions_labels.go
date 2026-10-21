package views

// The words of the actions: the bar and the menu word some of them differently, so each says its own.

// barLabel is the action as a button of the bulk bar says it.
func (m *Main) barLabel(a taskAction) string {
	switch a.Kind {
	case actDone:
		return m.say.ActionDone()
	case actMove:
		return m.say.MenuMoveTo()
	case actPriority:
		return m.priorityLabel(a)
	case actHandOff:
		return m.say.BulkHandOff()
	case actDelete:
		return m.say.MenuDelete()
	}
	return "" // Cancel has no button
}

// priorityLabel is Priority, or for the user's own tasks the star: Mark important, or Not important
// when every one has it.
func (m *Main) priorityLabel(a taskAction) string {
	switch {
	case !a.Star:
		return m.say.PropPriority()
	case a.Starred:
		return m.say.MenuNotImportant()
	}
	return m.say.MenuMarkImportant()
}

// menuLabel is the action as a line of the menu of the n selected tasks says it.
func (m *Main) menuLabel(a taskAction, n int) string {
	switch a.Kind {
	case actDone:
		return m.say.BulkMarkDone()
	case actCancel:
		return m.say.BulkCancel()
	case actMove:
		return m.say.MenuMoveTo()
	case actPriority:
		return m.priorityLabel(a)
	case actHandOff:
		return m.say.BulkHandOff()
	default:
		return m.say.BulkDeleteMenu(n)
	}
}

// actionSpec is the line of the menu for an action: its icon, hint and danger are the action's.
func actionSpec(a taskAction, keyPrefix, label string, run func()) popSpec {
	return popSpec{key: keyPrefix + a.Name, icon: a.Icon, label: label, hint: a.Key.hint(), danger: a.Danger, run: run}
}
