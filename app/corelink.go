package main

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/corelink"
)

// linkBundledCore points the adhd-core path the agents run at the copy inside the app, so that
// updating the app updates it. A build that runs outside an app bundle (go run, mygo dev) has no
// copy and does nothing. It logs what it did; nothing is shown.
func linkBundledCore(resources, link string) {
	if !strings.Contains(resources, ".app/") {
		return
	}
	res, err := corelink.Ensure(link, filepath.Join(resources, "adhd-core"))
	if err != nil {
		log.Printf("adhd-core link %s: %v", link, err)
		return
	}
	log.Printf("adhd-core link %s: %s (%s)", link, res.Action, res.Reason)
}
