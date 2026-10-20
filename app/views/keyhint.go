package views

import "strings"

// keyHintWords are the Mac key symbols and what the other systems write for them.
var keyHintWords = strings.NewReplacer("⌘", "Ctrl+", "⇧", "Shift+", "⌥", "Alt+", "↵", "Enter", "⌫", "Backspace")

// keyHint writes a shortcut for the system: the Mac's symbols as they are on a Mac (⌘⇧A), and as
// words elsewhere (Ctrl+Shift+A). The keys themselves are the same on both: ui.Cmd is Ctrl outside
// the Mac.
func keyHint(mac string) string {
	if systemGOOS == "darwin" {
		return mac
	}
	return keyHintWords.Replace(mac)
}
