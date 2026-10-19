package data

import (
	"context"
	"errors"

	"github.com/QingYunA/adhd-todo/internal/agent"
)

// The Recent changes page's requests to the assistant.

// RecentDescribeRequest is a repository's items to be put in a sentence each.
type RecentDescribeRequest struct {
	Repo string
	// Language is the language to write in, by name ("English", "Simplified Chinese").
	Language string
	Items    []RecentItem
}

// RecentLinesRequest is what a repository did in a period, for the sentences at the head of its
// section: its lines as they are listed, with the sentences the assistant or the titles gave them.
type RecentLinesRequest struct {
	Repo     string
	Language string
	// Period says which, as words ("the last 7 days").
	Period string
	Lines  []RecentLine
}

// RecentExplainRequest is one item to be explained.
type RecentExplainRequest struct {
	Language string
	Item     RecentItem
}

// DescribeRecent implements Assistant. The error is already in plain words.
func (a *AgentAssistant) DescribeRecent(ctx context.Context, req RecentDescribeRequest) (map[int]RecentNote, error) {
	in := agent.RecentDescribeInput{Repo: req.Repo, Language: req.Language}
	for _, i := range req.Items {
		in.Items = append(in.Items, briefOf(i))
	}
	notes, err := a.Agent.DescribeRecent(ctx, in)
	if err != nil {
		return nil, recentProblem(err)
	}
	out := make(map[int]RecentNote, len(notes))
	for n, note := range notes {
		out[n] = RecentNote{Sentence: note.Sentence, Category: RecentCategory(note.Category), Topic: note.Topic}
	}
	return out, nil
}

// WriteRecentLines implements Assistant. The error is already in plain words.
func (a *AgentAssistant) WriteRecentLines(ctx context.Context, req RecentLinesRequest) (string, error) {
	in := agent.RecentLinesInput{Repo: req.Repo, Language: req.Language, Period: req.Period}
	for _, l := range req.Lines {
		switch l.Item.Kind {
		case RecentMerged:
			in.Merged = append(in.Merged, agent.RecentLinesItem{Category: string(l.Note.Category), Sentence: l.Note.Sentence})
		case RecentOpen:
			in.Open = append(in.Open, l.Note.Sentence)
		case RecentIssue:
			in.Issues = append(in.Issues, agent.RecentLinesIssue{Sentence: l.Note.Sentence, Solved: l.Item.Solved()})
		}
	}
	text, err := a.Agent.WriteRecentLines(ctx, in)
	if err != nil {
		return "", recentProblem(err)
	}
	return text, nil
}

// ExplainRecent implements Assistant. The error is already in plain words.
func (a *AgentAssistant) ExplainRecent(ctx context.Context, req RecentExplainRequest) (RecentExplanation, error) {
	out, err := a.Agent.ExplainRecent(ctx, agent.RecentExplainInput{
		Repo: req.Item.Repo, Language: req.Language, Item: briefOf(req.Item), Files: req.Item.Files, FixedBy: req.Item.FixedBy,
	})
	if err != nil {
		return RecentExplanation{}, recentProblem(err)
	}
	return out, nil
}

// briefOf is an item as the model is told about it.
func briefOf(i RecentItem) agent.RecentItem {
	return agent.RecentItem{Number: i.Number, Kind: string(i.Kind), Title: i.Title, Labels: i.Labels, Body: i.Body}
}

// recentProblem is problemFor, leaving a cancelled request as it is.
func recentProblem(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	return problemFor(err)
}
