package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func svg(d string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="` + d + `"/></svg>`
}

// library writes a few icons: two that reduce exactly, one that is lossy, one with a curve, one sharp.
func library(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"square.svg":       svg("M2 2h4v4H2z"),
		"bar.svg":          svg("M4 2h2v8H4z"),
		"lossy.svg":        svg("M1 1h1v1H1zM5 1h1v1H5zM9 1h1v1H9zM1 5h1v1H1zM5 5h1v1H5zM9 5h1v1H9z"),
		"curve.svg":        svg("M2 2C4 4 6 6 8 8z"),
		"square-sharp.svg": svg("M2 2h4v4H2z"),
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func write(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseCuratedReadsGroupsKeywordsAndOverrides(t *testing.T) {
	rows := strings.Repeat("  ............\n", 11) + "  #...........\n"
	groups, err := parseCurated(strings.NewReader("# a comment\n\n[Work]\nsquare 方块  正方形\nbar\n\n[Home]\nmine 我的\n" + rows))
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0].Name != "Work" || len(groups[0].Icons) != 2 {
		t.Fatalf("%+v", groups)
	}
	if got := groups[0].Icons[0]; got.Name != "square" || got.Keywords != "方块 正方形" || got.Rows != nil {
		t.Fatalf("%+v", got)
	}
	if got := groups[1].Icons[0]; got.Name != "mine" || len(got.Rows) != 12 || got.Rows[11] != "#..........." {
		t.Fatalf("%+v", got)
	}
}

func TestParseCuratedRefusesBrokenFiles(t *testing.T) {
	for name, in := range map[string]string{
		"icon before group": "square\n",
		"short override":    "[A]\nsquare\n  ....\n",
		"duplicate icon":    "[A]\nsquare\n[B]\nsquare\n",
		"empty group":       "[A]\n[B]\nsquare\n",
	} {
		if _, err := parseCurated(strings.NewReader(in)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestGenerateKeepsTheCloseIconsAndReportsTheRest(t *testing.T) {
	lib := library(t)
	curated := write(t, "c.txt", "[Shapes]\nsquare 方块\nbar\n")
	out := filepath.Join(t.TempDir(), "gen.go")
	var log bytes.Buffer
	if err := run([]string{"-svg", lib, "-curated", curated, "-max-error", "4", "-o", out}, &log); err != nil {
		t.Fatalf("%v\n%s", err, log.String())
	}
	got, _ := os.ReadFile(out)
	s := string(got)
	for _, want := range []string{"DO NOT EDIT", `{"bar", "`, `{"square", "`, `"方块"`, `{"Shapes", []string{"square", "bar"}}`} {
		if !strings.Contains(s, want) {
			t.Errorf("output lacks %s:\n%s", want, s)
		}
	}
	for _, not := range []string{"curve", "sharp", `{"lossy"`} {
		if strings.Contains(s, not) {
			t.Errorf("output has %s", not)
		}
	}
	if !strings.Contains(log.String(), "dropped curve") || !strings.Contains(log.String(), "dropped lossy") {
		t.Errorf("log: %s", log.String())
	}
}

func TestGenerateRefusesACuratedIconThatDoesNotConvert(t *testing.T) {
	lib := library(t)
	curated := write(t, "c.txt", "[Shapes]\nsquare\nlossy\n")
	out := filepath.Join(t.TempDir(), "gen.go")
	var log bytes.Buffer
	err := run([]string{"-svg", lib, "-curated", curated, "-max-error", "4", "-o", out}, &log)
	if err == nil || !strings.Contains(err.Error(), "lossy") {
		t.Fatalf("err = %v", err)
	}
}

func TestAnOverrideReplacesTheLibraryIcon(t *testing.T) {
	lib := library(t)
	rows := "  #...........\n" + strings.Repeat("  ............\n", 11)
	curated := write(t, "c.txt", "[Shapes]\nlossy 有损\n"+rows)
	out := filepath.Join(t.TempDir(), "gen.go")
	var log bytes.Buffer
	if err := run([]string{"-svg", lib, "-curated", curated, "-max-error", "4", "-o", out}, &log); err != nil {
		t.Fatalf("%v\n%s", err, log.String())
	}
	got, _ := os.ReadFile(out)
	if !strings.Contains(string(got), `{"lossy", "800000000000000000000000000000000000", "有损"}`) {
		t.Errorf("override missing:\n%s", got)
	}
}
