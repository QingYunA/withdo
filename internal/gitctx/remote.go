package gitctx

import (
	"net/url"
	"regexp"
	"strings"
)

var githubHosts = map[string]bool{
	"github.com":     true,
	"www.github.com": true,
	"ssh.github.com": true,
}

// scpLike matches "[user@]host:path" (not "scheme://").
var scpLike = regexp.MustCompile(`^(?:[^@/\s]+@)?([^:/\s]+):(.+)$`)

// ParseRemoteURL turns a git remote URL into "owner/repo". Only GitHub
// remotes are recognised; anything else returns "".
func ParseRemoteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	host, path, ok := splitRemote(raw)
	if !ok || !githubHosts[strings.ToLower(host)] {
		return ""
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	return parts[0] + "/" + parts[1]
}

func splitRemote(raw string) (host, path string, ok bool) {
	if !strings.Contains(raw, "://") {
		m := scpLike.FindStringSubmatch(raw)
		if m == nil {
			return "", "", false
		}
		return m[1], m[2], true
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "", "", false
	}
	return u.Hostname(), u.Path, true
}
