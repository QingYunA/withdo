package agent

import (
	"context"
	"strings"
	"testing"
)

func recentItems() RecentDescribeInput {
	return RecentDescribeInput{
		Repo: "QingYunA/adhd-todo", Language: "Simplified Chinese",
		Items: []RecentItem{
			{Number: 41, Kind: RecentMerged, Title: "fix(summary): give a two-digit ledger count room", Body: "48 was cut to an ellipsis"},
			{Number: 39, Kind: RecentOpen, Title: "feat(categories): pixel icons"},
			{Number: 58, Kind: RecentIssue, Title: "Export fails with pictures", Labels: []string{"bug"}},
		},
	}
}

func TestDescribeRecentAsksOnceAndReadsASentenceAndACategoryForEachNumber(t *testing.T) {
	r := newRig(t, "UTC", oct14, say(`{"41": {"sentence": "周报总数是两位数时不再被截断", "category": "Fixed", "topic": "ignored"},
		"39": {"sentence": "分类有了像素图标", "category": "new"}, "58": {"sentence": "带图片的任务导出会出错", "category": "fixed"}, "99": {"sentence": "not given"}}`))
	got, err := r.agent.DescribeRecent(context.Background(), recentItems())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[41].Sentence != "周报总数是两位数时不再被截断" || got[41].Category != RecentFixed || got[41].Topic != "" {
		t.Errorf("got %+v", got)
	}
	if got[39].Category != "" || got[58].Category != "" {
		t.Errorf("only a merged pull request has a category: %+v", got)
	}
	sent := r.model.sent(0)
	contains(t, "prompt", sent[0].Content, "Write in Simplified Chinese")
	contains(t, "prompt", sent[0].Content, "At most 12 words in English, at most 30 characters in Chinese")
	contains(t, "items", sent[1].Content, "#41 | merged | fix(summary): give a two-digit ledger count room")
	contains(t, "items", sent[1].Content, "description: 48 was cut to an ellipsis")
	contains(t, "items", sent[1].Content, "#58 | issue | Export fails with pictures | labels bug")
}

func TestDescribeRecentKeepsTheTopicOfAnOtherAndFailsWithoutUsableJSON(t *testing.T) {
	in := RecentDescribeInput{Items: []RecentItem{{Number: 3, Kind: RecentMerged, Title: "docs: readme"}}}
	r := newRig(t, "UTC", oct14, say("```json\n{\"3\": {\"sentence\": \"README is clearer\", \"category\": \"other\", \"topic\": \" docs \"}}\n```"))
	got, err := r.agent.DescribeRecent(context.Background(), in)
	if err != nil || got[3].Category != RecentOther || got[3].Topic != "docs" {
		t.Errorf("got %+v, %v", got, err)
	}
	for _, reply := range []string{"nope", `{"3": {"sentence": ""}}`, `{"7": {"sentence": "x"}}`} {
		r := newRig(t, "UTC", oct14, say(reply))
		if _, err := r.agent.DescribeRecent(context.Background(), in); err == nil {
			t.Errorf("%q was accepted", reply)
		}
	}
	if _, err := r.agent.DescribeRecent(context.Background(), RecentDescribeInput{}); err == nil {
		t.Error("nothing to describe was accepted")
	}
}

func TestWriteRecentLinesToldTheSentencesAndNotTheCode(t *testing.T) {
	r := newRig(t, "UTC", oct14, say("  Two things went in. One bug is still open.  "))
	got, err := r.agent.WriteRecentLines(context.Background(), RecentLinesInput{
		Repo: "QingYunA/adhd-todo", Language: "English", Period: "the last 7 days",
		Merged: []RecentLinesItem{{Category: "fixed", Sentence: "Ledger counts fit"}},
		Open:   []string{"Category icons"},
		Issues: []RecentLinesIssue{{Sentence: "Export fails", Solved: true}},
	})
	if err != nil || got != "Two things went in. One bug is still open." {
		t.Fatalf("got %q, %v", got, err)
	}
	sent := r.model.sent(0)
	contains(t, "prompt", sent[0].Content, "Never invent")
	contains(t, "request", sent[1].Content, "- [fixed] Ledger counts fit")
	contains(t, "request", sent[1].Content, "- Category icons")
	contains(t, "request", sent[1].Content, "- [solved] Export fails")
}

func TestExplainRecentGivesWhatWhyWhereFromTheFileNames(t *testing.T) {
	r := newRig(t, "UTC", oct14, say(`{"what": "Counts fit.", "why": "48 was cut.", "where": "The ledger view."}`))
	got, err := r.agent.ExplainRecent(context.Background(), RecentExplainInput{
		Repo: "QingYunA/adhd-todo", Language: "English", Item: RecentItem{Number: 41, Kind: RecentMerged, Title: "fix ledger", Body: "48 cut"},
		Files: []string{"app/views/summary_ledger.go"},
	})
	if err != nil || got.What != "Counts fit." || got.Why != "48 was cut." || got.Where != "The ledger view." {
		t.Fatalf("got %+v, %v", got, err)
	}
	contains(t, "request", r.model.sent(0)[1].Content, "Changed files (1): app/views/summary_ledger.go")
	r = newRig(t, "UTC", oct14, say(`{"what": "", "why": "x"}`))
	if _, err := r.agent.ExplainRecent(context.Background(), RecentExplainInput{Item: RecentItem{Number: 1}}); err == nil || strings.Contains(err.Error(), "<nil>") {
		t.Errorf("an explanation without a what was accepted: %v", err)
	}
}
