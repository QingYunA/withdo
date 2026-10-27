package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/agentsetup"
	"github.com/QingYunA/adhd-todo/internal/corelink"
)

// prepareAgents makes the withdo path the agents run point at this app, then (when migrate is set,
// which the demo modes leave off: they must not touch the real agents) moves the agents that
// still have the pre-rename adhd-todo MCP entry over to withdo. The link comes first, since a
// migrated agent is pointed at a withdo that has to exist. It runs in the background and never
// asks anything: what it does or fails to do is logged.
func prepareAgents(env *agentsetup.Env, migrate bool) {
	linkOwnExecutable(env.CorePath())
	if !migrate {
		return
	}
	for _, m := range env.Migrate(context.Background()) {
		if m.Err != nil {
			log.Printf("agent %s: moving adhd-todo to withdo: %v", m.Agent.Name, m.Err)
			continue
		}
		log.Printf("agent %s: moved adhd-todo to withdo", m.Agent.Name)
	}
}

// legacyCoreName is what the command line was called before the rename to Withdo. Its link stays
// next to the new one, so that an agent config that still runs it keeps working.
const legacyCoreName = "adhd-core"

// linkOwnExecutable points the withdo path the agents run at this app's executable, which is
// withdo too, so that updating the app updates it, and the legacy adhd-core next to it. A build
// that runs outside an app bundle (go run, mygo dev) does nothing. It logs what it did; nothing
// is shown.
func linkOwnExecutable(link string) {
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		log.Printf("core link %s: %v", link, err)
		return
	}
	linkBundledCores(exe, link)
}

// linkBundledCores links the withdo path and the legacy adhd-core one beside it.
func linkBundledCores(exe, link string) {
	linkBundledCore(exe, link)
	linkBundledCore(exe, filepath.Join(filepath.Dir(link), legacyCoreName))
}

// linkBundledCore links link at exe when exe is inside an installed app bundle. A bundle run from
// a disk image or translocated by Gatekeeper is not: the link would outlive the path it points at.
func linkBundledCore(exe, link string) {
	if !strings.Contains(exe, ".app/") {
		return
	}
	if strings.HasPrefix(exe, "/Volumes/") || strings.Contains(exe, "/AppTranslocation/") {
		log.Printf("core link %s: not linking from a disk image or a translocated app (%s)", link, exe)
		return
	}
	res, err := corelink.Ensure(link, exe)
	if err != nil {
		log.Printf("core link %s: %v", link, err)
		return
	}
	log.Printf("core link %s: %s (%s)", link, res.Action, res.Reason)
}
