package data

import (
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/QingYunA/adhd-todo/app/locales"
)

// The Markdown of a Summary (ui-spec, Summary > Export Markdown): the paragraph first, then a
// section per person with a line per task, then what is still open. The desktop's export sheet and
// the phone's share sheet both write it from here.

// Markdown is the summary as Markdown, in the language of say. paragraph is the assistant's, "" for
// none; links finds the pages the reports of a task attached (TaskLinks), nil for none.
func (s Summary) Markdown(say locales.Messages, paragraph string, links func(id int) []Link) string {
	var b strings.Builder
	title := say.SummaryMdDay(s.Period.FirstDate())
	if s.Period.Week {
		title = say.SummaryMdWeek(s.Period.FirstDate(), s.Period.Day(6).Format("2006-01-02"))
	}
	b.WriteString("# " + title + "\n")
	if p := strings.TrimSpace(paragraph); p != "" {
		b.WriteString("\n" + p + "\n")
	}
	for _, p := range s.People {
		name := say.DoneByYou()
		if !p.You() {
			name = AgentName(p.Agent)
		}
		b.WriteString("\n## " + name + "\n")
		for _, task := range p.Tasks {
			b.WriteString("- " + markdownTask(say, task, links) + "\n")
		}
	}
	if open := OpenParts(say, s.Open); open != "" {
		b.WriteString("\n" + say.SummaryMdOpen(open) + "\n")
	}
	return b.String()
}

// markdownTask is a line of the export: `title (project, PR #12): what the agent last said`.
func markdownTask(say locales.Messages, task SummaryTask, links func(id int) []Link) string {
	var detail []string
	if task.Project != "" {
		detail = append(detail, task.Project)
	}
	if task.PRNumber > 0 {
		detail = append(detail, "PR #"+strconv.Itoa(task.PRNumber))
	}
	line := task.Title
	if len(detail) > 0 {
		line += say.SummaryMdParen(strings.Join(detail, ", "))
	}
	if task.Said != "" {
		line += say.SummaryMdSaid(task.Said)
	}
	if links != nil {
		for _, l := range links(task.ID) {
			line += " " + MarkdownLink(l)
		}
	}
	return line
}

// TaskLinks are the pages the reports of a task attached, oldest report first, each address once.
// reports are newest first, as the snapshot has them.
func TaskLinks(reports []Report) []Link {
	var out []Link
	seen := map[string]bool{}
	for i := len(reports) - 1; i >= 0; i-- {
		for _, l := range reports[i].Links {
			if !seen[l.URL] {
				seen[l.URL] = true
				out = append(out, l)
			}
		}
	}
	return out
}

// MarkdownLink is a page as `[title](url)`, with what would end either part escaped.
func MarkdownLink(l Link) string {
	title := strings.NewReplacer("[", "\\[", "]", "\\]").Replace(l.Label())
	addr := strings.NewReplacer(" ", "%20", "(", "%28", ")", "%29").Replace(l.URL)
	return "[" + title + "](" + addr + ")"
}

// Label is what a link chip says: the title the agent gave, else the file's name, else the host and
// the end of the path ("github.com › pull/19").
func (l Link) Label() string {
	if l.Title != "" {
		return l.Title
	}
	u, err := url.Parse(l.URL)
	if err != nil {
		return l.URL
	}
	if strings.EqualFold(u.Scheme, "file") {
		return path.Base(u.Path)
	}
	if u.Host == "" {
		return l.URL
	}
	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	if segs[0] == "" {
		return u.Host
	}
	return u.Host + " › " + strings.Join(segs[max(0, len(segs)-2):], "/")
}

// OpenParts says what is still open as "3 in progress, 1 blocked", or "" when nothing is.
func OpenParts(say locales.Messages, o SummaryOpen) string {
	var parts []string
	if o.InProgress > 0 {
		parts = append(parts, say.SummaryOpenProgress(o.InProgress))
	}
	if o.InReview > 0 {
		parts = append(parts, say.SummaryOpenReview(o.InReview))
	}
	if o.Blocked > 0 {
		parts = append(parts, say.SummaryOpenBlocked(o.Blocked))
	}
	return strings.Join(parts, say.NotifyCountSeparator())
}
