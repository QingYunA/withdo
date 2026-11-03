package quickadd

import (
	"reflect"
	"testing"
)

// The words that were read are pointed at in the typed text, in bytes of what was typed.
func TestSpansPointAtTheWordsRead(t *testing.T) {
	tests := []struct {
		name, in string
		read     []string // the text of each span
	}{
		{"plain", "买牛奶", nil},
		{"zh day and time", "明天9点交房租", []string{"明天9点"}},
		{"remind me", "remind me to call mom tomorrow at 9am", []string{"remind me to", "tomorrow", "at 9am"}},
		{"deadline words", "写周报 周五前", []string{"周五前"}},
		{"category", "买菜 #Home", []string{"#Home"}},
		{"important", "交报告 明天 !", []string{"明天", "!"}},
		{"nothing but a date", "明天", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			drafts := Parse(tc.in, wed, cats)
			if len(drafts) != 1 {
				t.Fatalf("drafts = %+v", drafts)
			}
			var got []string
			for _, sp := range drafts[0].Spans {
				got = append(got, tc.in[sp.Start:sp.End])
			}
			if !reflect.DeepEqual(got, tc.read) {
				t.Errorf("spans read %q, want %q", got, tc.read)
			}
		})
	}
}

// A second task of the sentence has its spans in the whole text, not in its own part.
func TestSpansOfASecondTaskCountFromTheStart(t *testing.T) {
	in := "买牛奶；后天体检"
	drafts := Parse(in, wed, cats)
	if len(drafts) != 2 || len(drafts[0].Spans) != 0 || len(drafts[1].Spans) != 1 {
		t.Fatalf("drafts = %+v", drafts)
	}
	sp := drafts[1].Spans[0]
	if got := in[sp.Start:sp.End]; got != "后天" {
		t.Errorf("span = %q, want 后天", got)
	}
}
