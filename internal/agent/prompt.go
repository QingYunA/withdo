package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// lookaheadDays is how many upcoming days the prompt spells out, so the model never has to work
// out a weekday or a daylight-saving offset by itself.
const lookaheadDays = 14

// BuildPrompt is the system prompt for one turn. The clock, weekday, time zone and project list
// are part of it because relative times ("明天九点", "tonight") are meaningless without them; so are
// the user's categories, which a task is sorted into by name. The scope says where the request was
// typed, and focus is the task of a task scope.
func BuildPrompt(now time.Time, projects []domain.Project, categories []domain.Category, sc Scope, focus *domain.Task) string {
	var b strings.Builder
	b.WriteString(preamble)
	fmt.Fprintf(&b, "\nNow: %s (%s, UTC%s). RFC 3339: %s\n",
		now.Format("Monday, 2006-01-02 15:04"), now.Location(), offsetLabel(now), now.Format(time.RFC3339))
	b.WriteString(upcomingDays(now))
	b.WriteString(offsetNote(now))
	b.WriteString("\n" + projectList(projects))
	b.WriteString(categoryList(categories))
	b.WriteString("\n" + scopeSection(now, projects, categories, sc, focus))
	b.WriteString(rules)
	return b.String()
}

const preamble = `You are the assistant inside adhd-todo, a todo app for a software developer who runs several coding agents in parallel and has ADHD. The user types short, loose requests into one input field. You turn them into task changes with your tools.
`

const rules = `
Rules:
- Prefer acting. Do not ask for confirmation; the user sees a receipt with Undo.
- Resolve relative times against Now and write absolute RFC 3339 times with a UTC offset in tool calls. A time without an offset is rejected.
- Default times: morning (早上, 上午, 明早) is 09:00, noon (中午) 12:00, afternoon (下午) 14:00, "after lunch" 13:30, "before I leave" (下班前) 18:00, evening (晚上) 19:00, "tonight" (今晚) 20:00, the weekend (周末) Saturday 10:00. A bare day ("tomorrow", "Friday", "明天") is 09:00 for a reminder and end of day, 18:00, for a deadline.
- Reminder or deadline: "remind me", "提醒我" make a task with remind_at ("提醒我开会" becomes the title "开会"); "by Friday", "due", "deadline", "before standup" make due_at. Use both when the words give both.
- Priority comes from the words. "urgent", "asap", "blocker", 紧急, 急, 尽快, 马上 is P1; "when I get to it", "someday", "whenever", 不急, 有空, 以后再说 is P3; P0 only for "drop everything", "production is down", 线上挂了, 生产事故. A priority the user writes ("p1", "P2") is taken as written. Otherwise leave it out: P2 is the default.
- Category is what a task is for the user: one of the categories above. Set category on every new task from its words: "review the PR", "写周报", "回复客户邮件" are Work; "pay rent", "交房租", "买牛奶", "约牙医" are Life; words that fit the name of another category go there. When the user did not name a category, choose the best-fitting one yourself in the same create_task call; when the user's own words name it ("放到生活里"), also pass category_named true. Leave it out when none fits, or when the task goes into a repository's project (that counts as Work). Change it with update_task when the user asks ("这个算工作"); clear it to unsort the task.
- One message may hold several things ("buy milk, answer Sam's email", "买牛奶，回邮件"): make separate tasks, one per thing, each with its own time and priority.
- "X is done", "做完了 X", "X 搞定了" mean complete_task on that task.
- Titles are short and start with the action ("Review PR 12", "回复设计反馈"); leave the time, priority and "remind me" words out of the title, they go into the fields.
- Look tasks up with find_tasks before you change one. Never invent task ids.
- Ask only when acting would be wrong without the answer, for example two tasks match, or a reminder has no time. Ask with ask_user, offering 2 to 4 short options, and stop: do not write anything else in that turn.
- Projects are optional. A new task goes where the scope above says; project_id 0 means the Inbox.
- Reply in the language the user wrote in, in one short sentence stating what you did and what you inferred, for example "Added it to adhd-todo, due Friday, P1." No markdown.
`

// imageRules tells the model about the pictures of this request: the tasks it creates or changes keep
// them. Without pictures the prompt says nothing about them.
func imageRules(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("- The request has %d picture(s), numbered 0 to %d in the order you see them. A task you create or change with create_task or update_task keeps pictures: "+
		"pass images with the numbers of the pictures that belong to that task (an empty list for none). Leave images out when you cannot tell which belong where: the task then keeps all of them.\n", n, n-1)
}

func offsetLabel(t time.Time) string {
	_, secs := t.Zone()
	sign := '+'
	if secs < 0 {
		sign, secs = '-', -secs
	}
	return fmt.Sprintf("%c%02d:%02d", sign, secs/3600, secs%3600/60)
}

// upcomingDays lists the next days with weekday and date, starting tomorrow.
func upcomingDays(now time.Time) string {
	days := make([]string, 0, lookaheadDays)
	for i := 1; i <= lookaheadDays; i++ {
		days = append(days, now.AddDate(0, 0, i).Format("Mon 2006-01-02"))
	}
	return "Next days: " + strings.Join(days, ", ") + ".\n"
}

// offsetNote says which UTC offset applies on which dates when daylight saving changes within
// the lookahead, and otherwise that it stays fixed.
func offsetNote(now time.Time) string {
	cur := offsetLabel(noon(now, 0))
	var changes []string
	for i := 1; i <= lookaheadDays; i++ {
		d := noon(now, i)
		if off := offsetLabel(d); off != cur {
			changes = append(changes, fmt.Sprintf("%s from %s", off, d.Format("2006-01-02")))
			cur = off
		}
	}
	if len(changes) == 0 {
		return fmt.Sprintf("The UTC offset stays %s for these days.\n", offsetLabel(now))
	}
	return fmt.Sprintf("The UTC offset is %s today and changes (daylight saving): %s. Use the offset of the date you write.\n",
		offsetLabel(noon(now, 0)), strings.Join(changes, ", "))
}

// noon is 12:00 local time i days from now, a moment daylight-saving changes never straddle.
func noon(now time.Time, i int) time.Time {
	y, m, d := now.AddDate(0, 0, i).Date()
	return time.Date(y, m, d, 12, 0, 0, 0, now.Location())
}

func projectList(projects []domain.Project) string {
	var b strings.Builder
	for _, p := range projects {
		if p.ArchivedAt == nil {
			fmt.Fprintf(&b, "- %d, %s\n", p.ID, p.Name)
		}
	}
	if b.Len() == 0 {
		return "Projects: none yet (everything goes to the Inbox).\n"
	}
	return "Projects (id, name):\n" + b.String()
}

// categoryList names the categories a task can be sorted into: Work and Life with the Chinese
// names the user may say them by, then the user's own.
func categoryList(categories []domain.Category) string {
	if len(categories) == 0 {
		return ""
	}
	names := make([]string, len(categories))
	for i, c := range categories {
		names[i] = c.Name
		switch c.Builtin {
		case domain.BuiltinWork:
			names[i] += " (工作)"
		case domain.BuiltinLife:
			names[i] += " (生活)"
		}
	}
	return "\nCategories (use these names, never invent one): " + strings.Join(names, ", ") + ".\n"
}
