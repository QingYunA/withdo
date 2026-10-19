package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/corelink"
)

// linkOwnExecutable points the adhd-core path the agents run at this app's executable, which is
// adhd-core too, so that updating the app updates it. A build that runs outside an app bundle
// (go run, mygo dev) does nothing. It logs what it did; nothing is shown.
func linkOwnExecutable(link string) {
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		log.Printf("adhd-core link %s: %v", link, err)
		return
	}
	linkBundledCore(exe, link)
}

// linkBundledCore links link at exe when exe is inside an app bundle.
func linkBundledCore(exe, link string) {
	if !strings.Contains(exe, ".app/") {
		return
	}
	res, err := corelink.Ensure(link, exe)
	if err != nil {
		log.Printf("adhd-core link %s: %v", link, err)
		return
	}
	log.Printf("adhd-core link %s: %s (%s)", link, res.Action, res.Reason)
}
