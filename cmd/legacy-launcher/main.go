// Command legacy-launcher is "ADHD Todo.exe" in a Windows install that was updated from the app's
// old name. The old version's updater swaps the files of the new archive into the old folder and
// then starts the executable it ran from, "ADHD Todo.exe", which is this program since the archive
// carries it under that name (scripts/build-release.sh). The Start menu and Desktop shortcuts of the
// old install point at it too. It starts Withdo.exe from its own folder with the same arguments and
// ends. Built with -H windowsgui: it opens no console and shows nothing.
package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// appExe is the executable the launcher stands in for.
const appExe = "Withdo.exe"

// target is the app next to the launcher at self.
func target(self string) string {
	return filepath.Join(filepath.Dir(self), appExe)
}

func main() {
	self, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command(target(self), os.Args[1:]...)
	detach(cmd)
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	_ = cmd.Process.Release()
}
