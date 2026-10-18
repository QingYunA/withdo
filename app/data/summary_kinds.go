package data

import (
	"strings"
	"unicode"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// GoodKinds are the goods a finished task can become in the Summary scene; the assistant picks one
// for each when it writes the paragraph.
var GoodKinds = domain.GoodKinds

// kindWords say which good the words of a task point to, in the order they are tried: a fix before a
// feature, a reply before a design. English words are whole words (also with s, es, ed, ing or d
// after them); Chinese ones are found anywhere in the text.
var kindWords = []struct {
	kind    string
	english []string
	chinese []string
}{
	{"jar", []string{"fix", "bug", "crash", "hotfix", "regression", "leak", "panic", "typo"}, []string{"修复", "修正", "崩溃", "报错", "缺陷", "空指针", "漏洞"}},
	{"flask", []string{"test", "tests", "e2e", "coverage", "benchmark"}, []string{"测试", "用例", "覆盖率", "压测"}},
	{"mail", []string{"reply", "respond", "email", "mail", "message", "answer", "feedback", "review", "comment"}, []string{"回复", "邮件", "消息", "反馈", "回信", "答复", "评论"}},
	{"gear", []string{"refactor", "migrate", "migrating", "migration", "port", "rename", "cleanup", "upgrade", "bump", "rewrite", "simplify"}, []string{"重构", "迁移", "升级", "改造", "整理", "清理", "重写", "拆分"}},
	{"coin", []string{"rent", "pay", "bill", "invoice", "tax", "refund", "budget", "salary", "payment", "expense"}, []string{"房租", "账单", "付款", "缴费", "报销", "发票", "工资", "还款", "转账", "水电"}},
	{"book", []string{"doc", "docs", "documentation", "readme", "design", "spec", "changelog", "notes", "write-up", "writeup", "guide"}, []string{"文档", "设计", "说明", "笔记", "稿", "手册", "周报", "日报"}},
	{"scroll", []string{"plan", "research", "investigate", "spike", "explore", "roadmap", "evaluate", "survey", "proposal"}, []string{"计划", "调研", "方案", "规划", "研究", "评估", "路线"}},
	{"gem", []string{"feat", "feature", "add", "implement", "build", "create", "support", "introduce"}, []string{"新增", "新功能", "功能", "实现", "支持", "增加", "接入", "上线"}},
}

// englishSuffixes may follow an English word of kindWords: "fixes", "tested", "porting".
var englishSuffixes = []string{"", "s", "es", "ed", "d", "ing"}

// GuessKind is the good a task becomes when the assistant did not pick one (no model is set, or the
// task was finished after the paragraph was written): the words of its title and of what is said
// about it (its branch, the pull request's title) point to one, else it is a crate.
func GuessKind(texts ...string) string {
	text := strings.ToLower(strings.Join(texts, " "))
	tokens := strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' })
	for _, k := range kindWords {
		for _, w := range k.chinese {
			if strings.Contains(text, w) {
				return k.kind
			}
		}
		if hasWord(tokens, k.english) {
			return k.kind
		}
	}
	return domain.GoodCrate
}

// hasWord reports whether a token is one of words, with an English ending or not. A branch name such
// as "feat/native-ui" is split at the slash and the dash.
func hasWord(tokens, words []string) bool {
	for _, tok := range tokens {
		for _, part := range strings.Split(tok, "-") {
			for _, w := range words {
				for _, suf := range englishSuffixes {
					if part == w+suf {
						return true
					}
				}
			}
		}
	}
	return false
}
