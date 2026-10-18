package domain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	// MaxReportLinks is how many pages one report can carry.
	MaxReportLinks = 5
	// MaxLinkTitleRunes bounds a link's title.
	MaxLinkTitleRunes = 80
)

// ReportLink is a page an agent attached to a report: an HTML report it wrote, a design draft.
// The app opens it in the default browser and never fetches it. Title may be empty.
type ReportLink struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// DroppedLink is a link a report did not keep, and why. It goes back to the agent, so Reason is
// written for it.
type DroppedLink struct {
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

// CleanLinks keeps the links a report can carry (https://, http:// with a host, file:// with an
// absolute path; a title of at most 80 characters; five at most) and says why it dropped each of
// the others. A bad link never fails a report. Kept links are trimmed and never nil.
func CleanLinks(in []ReportLink) (kept []ReportLink, dropped []DroppedLink) {
	kept = []ReportLink{}
	for _, l := range in {
		l.URL, l.Title = strings.TrimSpace(l.URL), strings.TrimSpace(l.Title)
		if reason := linkProblem(l); reason != "" {
			dropped = append(dropped, DroppedLink{URL: l.URL, Reason: reason})
		} else if len(kept) == MaxReportLinks {
			dropped = append(dropped, DroppedLink{URL: l.URL, Reason: fmt.Sprintf("at most %d links per report", MaxReportLinks)})
		} else {
			kept = append(kept, l)
		}
	}
	return kept, dropped
}

// linkProblem says what is wrong with l, or "" when it can be kept.
func linkProblem(l ReportLink) string {
	if l.URL == "" {
		return "the url is empty"
	}
	if len([]rune(l.Title)) > MaxLinkTitleRunes {
		return fmt.Sprintf("the title is longer than %d characters", MaxLinkTitleRunes)
	}
	u, err := url.Parse(l.URL)
	if err != nil {
		return "not a valid URL"
	}
	switch strings.ToLower(u.Scheme) {
	case "https", "http":
		if u.Host == "" {
			return "a web link needs a host"
		}
	case "file":
		if (u.Host != "" && u.Host != "localhost") || !strings.HasPrefix(u.Path, "/") {
			return "a file URL needs an absolute path, like file:///Users/me/report.html"
		}
	default:
		return "only https, http and file URLs"
	}
	return ""
}

// encodeLinks is the stored form of a report's links.
func encodeLinks(links []ReportLink) (string, error) {
	if links == nil {
		return "[]", nil
	}
	b, err := json.Marshal(links)
	if err != nil {
		return "", fmt.Errorf("encode links: %w", err)
	}
	return string(b), nil
}

// decodeLinks reads a report's stored links; a value that does not parse reads as none.
func decodeLinks(s string) []ReportLink {
	links := []ReportLink{}
	if s != "" && s != "[]" {
		if err := json.Unmarshal([]byte(s), &links); err != nil || links == nil {
			return []ReportLink{}
		}
	}
	return links
}
