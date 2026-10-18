package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/cloudwego/eino/schema"
)

// SummaryTask is one finished task as the model is told about it.
type SummaryTask struct {
	ID      int64
	Title   string
	Project string // empty for the Inbox
	Who     string // "you", or the agent's name
	PR      int    // the pull request number, 0 for none
	Merged  bool   // the pull request merged
	Said    string // what the last report says, or empty
}

// SummaryInput is what the assistant writes a summary of: the tasks finished in a period, and how
// many are still open.
type SummaryInput struct {
	Week bool
	// Label says which period it is, in words the model can use ("Friday 2026-10-17", "2026-10-13
	// to 2026-10-19").
	Label string
	// Language is the language to write in, by name ("English", "Simplified Chinese").
	Language string
	Tasks    []SummaryTask
	// InProgress, InReview and Blocked count the tasks that are still open.
	InProgress, InReview, Blocked int
}

// SummaryOutput is the paragraph and the kind of good each task became, by task id. A task the model
// gave no valid kind for is left out.
type SummaryOutput struct {
	Paragraph string
	Kinds     map[int64]string
}

const summaryRules = `You write the paragraph of the end-of-day or end-of-week summary inside adhd-todo, a todo app for a software developer who runs several coding agents in parallel.
You are given the tasks finished in the period: who finished each one, its project and pull request, and what the last report said. You are also told how many tasks are still open.

The paragraph:
- Two or three sentences for a day, three to five for a week. Plain text: no markdown, no lists, no emoji.
- Say what the user and each agent finished, which pull requests went in, and what is still waiting (in progress, in review, blocked).
- Use only the tasks you are given. Never invent a task, a number, a pull request or a reason. Call the user "you".
- Do not cheer, praise, preach or advise. State what happened, nothing more.

The kinds: say what each finished task became in the picture, one of these and nothing else:
gem (a new feature), jar (a bug fix), gear (a refactor or a migration), book (docs or design), flask (tests), mail (a reply or a message), coin (money, a bill, rent), scroll (a plan or research), crate (anything else).

Answer with one JSON object and nothing else: {"paragraph": "...", "kinds": {"<task id>": "<kind>"}}. Give a kind for every task id you were given.
`

// summaryPrompt is the system prompt of one summary.
func summaryPrompt(in SummaryInput) string {
	lang := in.Language
	if lang == "" {
		lang = "English"
	}
	return summaryRules + "Write the paragraph in " + lang + ".\n"
}

// summaryRequest is what the model is told about the period.
func summaryRequest(in SummaryInput) string {
	var b strings.Builder
	kind := "a day"
	if in.Week {
		kind = "a week"
	}
	fmt.Fprintf(&b, "Period: %s (%s).\nFinished (%d):\n", in.Label, kind, len(in.Tasks))
	for _, t := range in.Tasks {
		fmt.Fprintf(&b, "- #%d | %s | by %s", t.ID, oneLine(t.Title), t.Who)
		if t.Project != "" {
			b.WriteString(" | project " + oneLine(t.Project))
		}
		switch {
		case t.PR > 0 && t.Merged:
			fmt.Fprintf(&b, " | PR #%d merged", t.PR)
		case t.PR > 0:
			fmt.Fprintf(&b, " | PR #%d", t.PR)
		}
		if t.Said != "" {
			b.WriteString(" | last report: " + oneLine(t.Said))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Still open: %d in progress, %d in review, %d blocked.\n", in.InProgress, in.InReview, in.Blocked)
	return b.String()
}

// oneLine puts text on one line and keeps it short: a report can be long and the model only needs
// its gist.
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 240 {
		return string(r[:240]) + "…"
	}
	return s
}

// Summarize asks the model, the one the composer uses, for the paragraph of a period and the goods
// of its tasks. It makes one call with no tools and remembers nothing. It blocks, so call it off the
// UI thread.
func (a *Agent) Summarize(ctx context.Context, in SummaryInput) (SummaryOutput, error) {
	if len(in.Tasks) == 0 {
		return SummaryOutput{}, errors.New("nothing to summarize")
	}
	m, err := a.cfg.Model(ctx)
	if err != nil {
		return SummaryOutput{}, err
	}
	reply, err := m.Generate(ctx, []*schema.Message{
		schema.SystemMessage(summaryPrompt(in)),
		schema.UserMessage(summaryRequest(in)),
	})
	if err != nil {
		return SummaryOutput{}, fmt.Errorf("summary: %w", err)
	}
	return parseSummary(reply.Content, in)
}

// parseSummary reads the model's JSON, which may come wrapped in a code fence or a sentence. Kinds
// for tasks that were not given, and kinds that are not goods, are dropped.
func parseSummary(text string, in SummaryInput) (SummaryOutput, error) {
	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return SummaryOutput{}, errors.New("summary: the model did not answer with JSON")
	}
	var raw struct {
		Paragraph string         `json:"paragraph"`
		Kinds     map[string]any `json:"kinds"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &raw); err != nil {
		return SummaryOutput{}, fmt.Errorf("summary: %w", err)
	}
	para := strings.TrimSpace(raw.Paragraph)
	if para == "" {
		return SummaryOutput{}, errors.New("summary: the model wrote no paragraph")
	}
	given := map[int64]bool{}
	for _, t := range in.Tasks {
		given[t.ID] = true
	}
	out := SummaryOutput{Paragraph: para, Kinds: map[int64]string{}}
	for key, v := range raw.Kinds {
		id, err := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(key), "#"), 10, 64)
		kind, _ := v.(string)
		kind = strings.ToLower(strings.TrimSpace(kind))
		if err == nil && given[id] && domain.IsGoodKind(kind) {
			out.Kinds[id] = kind
		}
	}
	return out, nil
}
