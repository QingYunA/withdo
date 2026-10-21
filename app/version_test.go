package main

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/mcpserver"
)

// A release bumps the version by hand in four places; they must agree.
func TestVersionsAgree(t *testing.T) {
	raw, err := os.ReadFile("mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	gradle, err := os.ReadFile("../android/app/build.gradle.kts")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`versionName\s*=\s*"([^"]*)"`).FindSubmatch(gradle)
	if m == nil {
		t.Fatal("no versionName in android/app/build.gradle.kts")
	}
	got := map[string]string{
		"app/mygo.json version":                    cfg.Version,
		"internal/mcpserver/server.go Version":     mcpserver.Version,
		"android/app/build.gradle.kts versionName": string(m[1]),
	}
	for name, v := range got {
		if v != version {
			t.Errorf("%s is %q but app/main.go version is %q: bump all four together", name, v, version)
		}
	}
}
