package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// The Recent changes page asks the model three things, each in one call with no tools: a sentence
// and a category for every pull request and issue of a repository, two or three sentences about
// the repository's period, and what one item did. Like Summarize they remember nothing.

// Kinds of a RecentItem.
const (
	RecentMerged = "merged" // a pull request that merged
	RecentOpen   = "open"   // a pull request still open
	RecentIssue  = "issue"
)

// Categories a merged pull request falls in.
const (
	RecentNew   = "new"
	RecentFixed = "fixed"
	RecentOther = "other"
)

// RecentItem is a pull request or an issue as the model is told about it.
type RecentItem struct {
	Number int
	Kind   string // RecentMerged, RecentOpen or RecentIssue
	Title  string
	Labels []string
	Body   string // the description, already cut to the length the model is given
}

// RecentDescribeInput is one repository's items to be put in a sentence each.
type RecentDescribeInput struct {
	Repo string
	// Language is the language to write in, by name ("English", "Simplified Chinese").
	Language string
	Items    []RecentItem
}

// RecentNote is the sentence written for one item, with, for a merged pull request, its category
// and, for a category of other, a word or two naming what it is.
type RecentNote struct {
	Sentence string
	Category string
	Topic    string
}

const recentDescribeRules = `You write the lines of the Recent changes page inside Withdo, a todo app for a software developer who runs several coding agents in parallel.
You are given the pull requests and issues of one repository: each with its number, kind, title, labels and description.

For each one write a sentence saying plainly what it does for the user of the software, not how it is done:
- At most 12 words in English, at most 30 characters in Chinese. Plain text: no markdown, no emoji, no commit prefix such as "feat(scope):", no "this PR".
- Use only what you are given. Never invent a feature, a reason or a number.

For a merged pull request also give its category: new (a new feature or ability), fixed (a bug fix), or other (docs, refactors, dependencies, versions, tests, screenshots, chores). For other give a topic too: one or two words saying what it is ("docs", "version", "screenshots"), in the language you write in. Open pull requests and issues get no category and no topic.

Answer with one JSON object and nothing else, keyed by number: {"41": {"sentence": "...", "category": "fixed", "topic": ""}}. Answer for every number you were given.
`

// DescribeRecent asks for a sentence and a category for each item. An item the model gave no
// usable sentence for is left out of the result. It blocks, so call it off the UI thread.
func (a *Agent) DescribeRecent(ctx context.Context, in RecentDescribeInput) (map[int]RecentNote, error) {
	if len(in.Items) == 0 {
		return nil, errors.New("nothing to describe")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Repository: %s\nItems (%d):\n", in.Repo, len(in.Items))
	for _, it := range in.Items {
		fmt.Fprintf(&b, "- #%d | %s | %s", it.Number, it.Kind, oneLine(it.Title))
		if len(it.Labels) > 0 {
			b.WriteString(" | labels " + strings.Join(it.Labels, ", "))
		}
		if body := strings.TrimSpace(it.Body); body != "" {
			b.WriteString("\n  description: " + clip(body, 2000))
		}
		b.WriteString("\n")
	}
	text, err := a.once(ctx, recentDescribeRules+"Write in "+languageOr(in.Language)+".\n", b.String())
	if err != nil {
		return nil, err
	}
	return parseRecentNotes(text, in)
}

// parseRecentNotes reads the model's JSON, which may come wrapped in a code fence or a sentence.
func parseRecentNotes(text string, in RecentDescribeInput) (map[int]RecentNote, error) {
	var raw map[string]struct {
		Sentence string `json:"sentence"`
		Category string `json:"category"`
		Topic    string `json:"topic"`
	}
	if err := readJSON(text, &raw); err != nil {
		return nil, fmt.Errorf("changes: %w", err)
	}
	kinds := map[int]string{}
	for _, it := range in.Items {
		kinds[it.Number] = it.Kind
	}
	out := map[int]RecentNote{}
	for key, v := range raw {
		n, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(key), "#"))
		kind, given := kinds[n]
		sentence := strings.TrimSpace(v.Sentence)
		if err != nil || !given || sentence == "" {
			continue
		}
		note := RecentNote{Sentence: sentence}
		if kind == RecentMerged {
			switch c := strings.ToLower(strings.TrimSpace(v.Category)); c {
			case RecentNew, RecentFixed, RecentOther:
				note.Category = c
			}
			if note.Category == RecentOther {
				note.Topic = strings.TrimSpace(v.Topic)
			}
		}
		out[n] = note
	}
	if len(out) == 0 {
		return nil, errors.New("changes: the model wrote no sentence")
	}
	return out, nil
}

// RecentLinesInput is what a repository gained, fixed and still has open in a period, as sentences.
type RecentLinesInput struct {
	Repo     string
	Language string
	// Period says which, as words ("the last 7 days").
	Period string
	// Merged are the merged pull requests' sentences, with their category; Open the open pull
	// requests'; Issues the issues', each saying whether it was solved.
	Merged []RecentLinesItem
	Open   []string
	Issues []RecentLinesIssue
}

// RecentLinesItem is a merged pull request in a RecentLinesInput.
type RecentLinesItem struct{ Category, Sentence string }

// RecentLinesIssue is an issue in a RecentLinesInput.
type RecentLinesIssue struct {
	Sentence string
	Solved   bool
}

const recentLinesRules = `You write two or three sentences at the head of one project's section on the Recent changes page inside Withdo, a todo app for a software developer who runs several coding agents in parallel.
You are given what the repository merged in a period (each with its category), the pull requests still open, and its issues.

- Two or three sentences, plain text: no markdown, no lists, no emoji.
- Say what the project gained, what was fixed and what is still open. Leave out what there is nothing of.
- Use only what you are given. Never invent a feature, a number or a reason. Call nobody by name.
- Do not cheer, praise, preach or advise. State what happened, nothing more.
`

// WriteRecentLines asks for the two or three sentences about a repository's period. It blocks, so
// call it off the UI thread.
func (a *Agent) WriteRecentLines(ctx context.Context, in RecentLinesInput) (string, error) {
	if len(in.Merged)+len(in.Open)+len(in.Issues) == 0 {
		return "", errors.New("nothing to write about")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Repository: %s\nPeriod: %s\n", in.Repo, in.Period)
	fmt.Fprintf(&b, "Merged (%d):\n", len(in.Merged))
	for _, m := range in.Merged {
		fmt.Fprintf(&b, "- [%s] %s\n", m.Category, oneLine(m.Sentence))
	}
	fmt.Fprintf(&b, "Open pull requests (%d):\n", len(in.Open))
	for _, s := range in.Open {
		fmt.Fprintf(&b, "- %s\n", oneLine(s))
	}
	fmt.Fprintf(&b, "Issues (%d):\n", len(in.Issues))
	for _, i := range in.Issues {
		state := "open"
		if i.Solved {
			state = "solved"
		}
		fmt.Fprintf(&b, "- [%s] %s\n", state, oneLine(i.Sentence))
	}
	text, err := a.once(ctx, recentLinesRules+"Write in "+languageOr(in.Language)+".\n", b.String())
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("changes: the model wrote nothing")
	}
	return text, nil
}

// RecentExplainInput is one item to be explained: its description and the names of the files it changed.
type RecentExplainInput struct {
	Repo     string
	Language string
	Item     RecentItem
	// Files are the names of the changed files of a pull request, never their content.
	Files []string
	// FixedBy is the pull request that closed an issue, 0 for none.
	FixedBy int
}

// RecentExplanation is what an item did, why, and where it touched the code. An issue has no Where.
type RecentExplanation struct {
	What  string `json:"what"`
	Why   string `json:"why"`
	Where string `json:"where"`
}

const recentExplainRules = `You explain one pull request or issue to the developer who owns the project, inside Withdo.
You are given its title, its description and, for a pull request, the names of the files it changed. You never see the code.

Answer with three short points, each one or two plain sentences with no markdown:
- what: what it does.
- why: why it was done, from the description. If the description does not say, say what problem it addresses without guessing a motive.
- where: which parts of the code it touched, said in words from the file names (not a list of paths). Leave it empty for an issue.

Use only what you are given. Never invent. Answer with one JSON object and nothing else: {"what": "...", "why": "...", "where": "..."}.
`

// ExplainRecent asks what an item did, why and where. It blocks, so call it off the UI thread.
func (a *Agent) ExplainRecent(ctx context.Context, in RecentExplainInput) (RecentExplanation, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Repository: %s\n%s #%d: %s\n", in.Repo, in.Item.Kind, in.Item.Number, oneLine(in.Item.Title))
	if len(in.Item.Labels) > 0 {
		b.WriteString("Labels: " + strings.Join(in.Item.Labels, ", ") + "\n")
	}
	if in.FixedBy > 0 {
		fmt.Fprintf(&b, "Closed by pull request #%d.\n", in.FixedBy)
	}
	b.WriteString("Description:\n" + clip(strings.TrimSpace(in.Item.Body), 2000) + "\n")
	if len(in.Files) > 0 {
		files := in.Files
		if len(files) > 60 {
			files = files[:60]
		}
		fmt.Fprintf(&b, "Changed files (%d): %s\n", len(in.Files), strings.Join(files, ", "))
	}
	text, err := a.once(ctx, recentExplainRules+"Write in "+languageOr(in.Language)+".\n", b.String())
	if err != nil {
		return RecentExplanation{}, err
	}
	var raw RecentExplanation
	if err := readJSON(text, &raw); err != nil {
		return RecentExplanation{}, fmt.Errorf("changes: %w", err)
	}
	out := RecentExplanation{What: strings.TrimSpace(raw.What), Why: strings.TrimSpace(raw.Why), Where: strings.TrimSpace(raw.Where)}
	if out.What == "" || out.Why == "" {
		return RecentExplanation{}, errors.New("changes: the model gave no explanation")
	}
	return out, nil
}

// once makes one call to the model of the composer with a system prompt and one request.
func (a *Agent) once(ctx context.Context, system, request string) (string, error) {
	m, err := a.cfg.Model(ctx)
	if err != nil {
		return "", err
	}
	reply, err := m.Generate(ctx, []*schema.Message{schema.SystemMessage(system), schema.UserMessage(request)})
	if err != nil {
		return "", fmt.Errorf("changes: %w", err)
	}
	return reply.Content, nil
}

// readJSON decodes the one JSON object in text, which may be wrapped in a code fence or a sentence.
func readJSON(text string, v any) error {
	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return errors.New("the model did not answer with JSON")
	}
	return json.Unmarshal([]byte(text[start:end+1]), v)
}

func languageOr(lang string) string {
	if lang == "" {
		return "English"
	}
	return lang
}

// clip cuts text to at most n characters.
func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
