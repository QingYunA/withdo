package main

import (
	"log"
	"os"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// newWindowsInstall describes the running app: where the Start menu and Desktop shortcuts are
// (the folders the installer's $SMPROGRAMS and $DESKTOP are, wherever the user moved them) and its
// uninstall entry in HKCU.
func newWindowsInstall() (windowsInstall, bool) {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("legacy install: %v", err)
		return windowsInstall{}, false
	}
	w := windowsInstall{exe: exe, exists: pathExists, rename: os.Rename}
	w.startMenu, _ = windows.KnownFolderPath(windows.FOLDERID_Programs, 0)
	w.desktop, _ = windows.KnownFolderPath(windows.FOLDERID_Desktop, 0)
	w.getString = func(name string) (string, error) {
		k, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.QUERY_VALUE)
		if err != nil {
			return "", err
		}
		defer k.Close()
		v, _, err := k.GetStringValue(name)
		return v, err
	}
	w.setString = func(name, value string) error {
		k, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.SET_VALUE)
		if err != nil {
			return err
		}
		defer k.Close()
		return k.SetStringValue(name, value)
	}
	return w, true
}
