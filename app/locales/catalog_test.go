package locales

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"
)

// messages reads a catalog: its messages without the @descriptions and @@locale.
func messages(t *testing.T, file string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(raw, &all); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for k, v := range all {
		if strings.HasPrefix(k, "@") {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			t.Fatalf("%s: %s is not a string", file, k)
		}
		out[k] = s
	}
	return out
}

// A placeholder is a name after a brace; the branches of a select (`1{January}`) are text, and a word
// character before their brace tells them apart.
var placeholder = regexp.MustCompile(`(?:^|[^\w])\{(\w+)`)

func placeholders(msg string) []string {
	var names []string
	for _, m := range placeholder.FindAllStringSubmatch(msg, -1) {
		if !slices.Contains(names, m[1]) {
			names = append(names, m[1])
		}
	}
	slices.Sort(names)
	return names
}

func TestEveryEnglishMessageIsTranslatedToChinese(t *testing.T) {
	en, zh := messages(t, "en.json"), messages(t, "zh-Hans.json")
	for k, msg := range en {
		tr, ok := zh[k]
		switch {
		case !ok:
			t.Errorf("%s has no zh-Hans message", k)
		case strings.TrimSpace(tr) == "":
			t.Errorf("%s is empty in zh-Hans", k)
		case !slices.Equal(placeholders(msg), placeholders(tr)):
			t.Errorf("%s: placeholders %v in en, %v in zh-Hans", k, placeholders(msg), placeholders(tr))
		}
	}
	for k := range zh {
		if _, ok := en[k]; !ok {
			t.Errorf("%s is in zh-Hans only", k)
		}
	}
}

func TestEveryEnglishMessageIsDescribed(t *testing.T) {
	raw, err := os.ReadFile("en.json")
	if err != nil {
		t.Fatal(err)
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(raw, &all); err != nil {
		t.Fatal(err)
	}
	for k := range messages(t, "en.json") {
		if _, ok := all["@"+k]; !ok {
			t.Errorf("%s has no @%s description for the translators", k, k)
		}
	}
}

// Chinese is set with full-width punctuation, not the ASCII marks of the English next to it.
func TestChineseUsesFullWidthPunctuation(t *testing.T) {
	ascii := regexp.MustCompile(`[,;:!?()]`)
	for k, msg := range messages(t, "zh-Hans.json") {
		var han bool
		for _, r := range msg {
			han = han || unicode.Is(unicode.Han, r)
		}
		if !han {
			continue
		}
		for _, loc := range ascii.FindAllStringIndex(msg, -1) {
			// An ASCII mark inside a Latin run (a URL, "PR #12") is fine; one next to Han is not.
			before, after := msg[:loc[0]], msg[loc[1]:]
			if endsWithHan(before) || startsWithHan(after) {
				t.Errorf("%s: %q has an ASCII %q next to Chinese", k, msg, msg[loc[0]:loc[1]])
			}
		}
	}
}

func endsWithHan(s string) bool {
	r := []rune(s)
	return len(r) > 0 && unicode.Is(unicode.Han, r[len(r)-1])
}

func startsWithHan(s string) bool {
	for _, r := range s {
		return unicode.Is(unicode.Han, r)
	}
	return false
}
