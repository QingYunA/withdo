package quickadd

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// scan is the text of one task while its recognized words are cut out of it one by one.
type scan struct{ s string }

// find is the first match of re that ok accepts (any, when ok is nil), as submatch indexes.
func (sc *scan) find(re *regexp.Regexp, ok func(m []int) bool) []int {
	for _, m := range re.FindAllStringSubmatchIndex(sc.s, -1) {
		if ok == nil || ok(m) {
			return m
		}
	}
	return nil
}

// groups are the submatches of m as text; a group that did not take part is "".
func (sc *scan) groups(m []int) []string {
	g := make([]string, len(m)/2)
	for i := range g {
		if m[2*i] >= 0 {
			g[i] = sc.s[m[2*i]:m[2*i+1]]
		}
	}
	return g
}

// cut takes s[a:b] out, leaving a space where words were on both sides, and nothing between two
// Chinese characters, which are written without spaces.
func (sc *scan) cut(a, b int) {
	l, _ := utf8.DecodeLastRuneInString(sc.s[:a])
	r, _ := utf8.DecodeRuneInString(sc.s[b:])
	glue := " "
	if unicode.Is(unicode.Han, l) && unicode.Is(unicode.Han, r) {
		glue = ""
	}
	sc.s = sc.s[:a] + glue + sc.s[b:]
}

var (
	// before the words of a day or time: "by", "截止"... name a deadline, "on", "at" do not.
	preMark = regexp.MustCompile(`(?i)(?:\b(before|by|due on|due|until|till|on|at)|(截止到|截止于|截止|截至|最晚)|(@))\s*$`)
	// after them: "前" (before), as in 周五前.
	postMark = regexp.MustCompile(`^\s*(?:之前|以前|前|截止|截至)`)
)

// notAMark are the characters 前 begins a word with, as in 前台: those are not "before".
const notAMark = "台天面边后年端进往门列任男女"

// cutWords takes out the day or time at s[a:b] with the marker words around it, and says whether
// they made it a deadline.
func (sc *scan) cutWords(a, b int) (deadline bool) {
	if m := preMark.FindStringSubmatchIndex(sc.s[:a]); m != nil {
		word := strings.ToLower(strings.TrimSpace(sc.s[m[0]:m[1]]))
		a, deadline = m[0], word != "@" && word != "on" && word != "at"
	}
	if m := postMark.FindStringIndex(sc.s[b:]); m != nil {
		next, _ := utf8.DecodeRuneInString(sc.s[b+m[1]:])
		if !strings.ContainsRune(notAMark, next) {
			b += m[1]
			deadline = true
		}
	}
	sc.cut(a, b)
	return deadline
}

// category finds a "#name" that is one of categories (any case) and cuts it out; it returns the
// name as the list spells it. A "#" inside a word ("C#") or a name that is only the start of a
// longer word ("#homepage" for Home) is left alone.
func (sc *scan) category(categories []string) string {
	names := append([]string(nil), categories...)
	sort.SliceStable(names, func(i, j int) bool { return utf8.RuneCountInString(names[i]) > utf8.RuneCountInString(names[j]) })
	for i := 0; i < len(sc.s); i++ {
		if sc.s[i] != '#' {
			continue
		}
		if p, _ := utf8.DecodeLastRuneInString(sc.s[:i]); asciiWord(p) {
			continue
		}
		rest := []rune(sc.s[i+1:])
		for _, name := range names {
			n := utf8.RuneCountInString(name)
			if name == "" || n > len(rest) || !strings.EqualFold(string(rest[:n]), name) {
				continue
			}
			if last, _ := utf8.DecodeLastRuneInString(name); asciiWord(last) && n < len(rest) && asciiWord(rest[n]) {
				continue
			}
			sc.cut(i, i+1+len(string(rest[:n])))
			return name
		}
	}
	return ""
}

// asciiWord is a letter, digit or underscore of ASCII.
func asciiWord(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}
