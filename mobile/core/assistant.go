package core

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/agent"
	"github.com/QingYunA/adhd-todo/internal/providers"
)

// assistantMemoryFile is the assistant's notes about the user, in the app's data folder.
const assistantMemoryFile = "assistant-memory.md"

// assistant is the built-in assistant of the phone: one agent over the phone's database, the
// official model's state and the turn in flight.
type assistant struct {
	official officialState

	mu     sync.Mutex
	agent  *agent.Agent
	loc    *time.Location // the zone of the latest Ask: the agent's clock reads it
	cancel context.CancelFunc
}

// askPlace is the place a request was typed, which Ask reads:
//
//	{"kind":"project","project_id":3,"zone":"Asia/Shanghai","offset":28800}
//
// kind is today (the Today page, category_id the chip narrowing it, 0 for all), inbox, project
// (project_id), task (task_id) or calendar (day, "2006-01-02" in the user's zone). zone is the
// phone's IANA zone and offset its UTC offset in seconds now, the fallback when the core's Go has
// no zone database; without both the clock is the core's own.
type askPlace struct {
	Kind       string `json:"kind"`
	ProjectID  int64  `json:"project_id"`
	TaskID     int64  `json:"task_id"`
	Day        string `json:"day"`
	CategoryID int64  `json:"category_id"`
	Zone       string `json:"zone"`
	Offset     int    `json:"offset"`
}

func (p askPlace) scope() agent.Scope {
	switch p.Kind {
	case "project":
		return agent.Scope{Kind: agent.ScopeProject, ProjectID: p.ProjectID}
	case "task":
		return agent.Scope{Kind: agent.ScopeTask, TaskID: p.TaskID}
	case "calendar":
		return agent.Scope{Kind: agent.ScopeCalendar, Day: p.Day}
	case "inbox":
		return agent.Scope{Kind: agent.ScopeInbox}
	}
	return agent.Scope{Kind: agent.ScopeNow, CategoryID: p.CategoryID}
}

// location is the phone's zone, or nil to keep the core's clock.
func (p askPlace) location() *time.Location {
	if p.Zone == "" {
		return nil
	}
	if loc, err := time.LoadLocation(p.Zone); err == nil {
		return loc
	}
	return time.FixedZone(p.Zone, p.Offset)
}

// replyJSON is Ask's answer.
//
//	{"text":"","receipts":[{"id":1,"task_id":7,"title":"打给房东","status":"todo","bell":true,
//	   "detail":{"kind":"reminder","at":"2026-11-04T09:00:00+08:00"}}],
//	 "question":{"text":"哪个项目?","options":["withdo","other"]},"problem":""}
type replyJSON struct {
	Text     string        `json:"text"`
	Receipts []receiptJSON `json:"receipts"`
	Question *questionJSON `json:"question,omitempty"`
	// Problem is why the model did not answer, when it did not: quota, signed_out, off, cancelled,
	// busy, timeout, unreachable, rate_limited or other. Receipts made before the failure are still
	// listed, so they can be undone.
	Problem string `json:"problem,omitempty"`
	// Error is the English sentence for a problem, for logs; the app words the kinds itself.
	Error string `json:"error,omitempty"`
}

type questionJSON struct {
	Text    string   `json:"text"`
	Options []string `json:"options"`
}

// receiptJSON is one change the assistant made. status is the task's status after it, bell says a
// reminder was set or moved.
type receiptJSON struct {
	ID     int64      `json:"id"`
	TaskID int64      `json:"task_id"`
	Title  string     `json:"title"`
	Status string     `json:"status"`
	Bell   bool       `json:"bell"`
	Detail detailJSON `json:"detail"`
}

// detailJSON is what a receipt says, as data (the app words it): kind is added, reminder (at),
// due (at), updated (changes), done, remembered or forgot (title is the note). due is a deadline
// that came with a reminder; project is the project's name, "" for the Inbox; category is
// "work" or "life" for the built-in ones, else the user's own name; auto_filed says the assistant
// chose the category itself.
type detailJSON struct {
	Kind      string       `json:"kind"`
	At        string       `json:"at,omitempty"`
	Due       string       `json:"due,omitempty"`
	Project   string       `json:"project,omitempty"`
	Priority  string       `json:"priority,omitempty"`
	Category  string       `json:"category,omitempty"`
	AutoFiled bool         `json:"auto_filed,omitempty"`
	Changes   []changeJSON `json:"changes,omitempty"`
}

// changeJSON is one field an update changed: kind is reminder or due (at, "" when cleared), status,
// priority, project, category (value, "" when cleared), renamed or notes.
type changeJSON struct {
	Kind  string `json:"kind"`
	At    string `json:"at,omitempty"`
	Value string `json:"value,omitempty"`
}

var detailKinds = map[agent.DetailKind]string{
	agent.DetailAdded: "added", agent.DetailReminder: "reminder", agent.DetailDue: "due",
	agent.DetailUpdated: "updated", agent.DetailDone: "done",
	agent.DetailRemembered: "remembered", agent.DetailForgot: "forgot",
}

var changeKinds = map[agent.ChangeKind]string{
	agent.ChangeReminder: "reminder", agent.ChangeDue: "due", agent.ChangeStatus: "status",
	agent.ChangePriority: "priority", agent.ChangeProject: "project", agent.ChangeRenamed: "renamed",
	agent.ChangeNotes: "notes", agent.ChangeCategory: "category",
}

func zoneText(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func receiptOf(r agent.Receipt) receiptJSON {
	d := r.Detail
	out := receiptJSON{
		ID: int64(r.ID), TaskID: r.TaskID, Title: r.Title, Status: r.Status, Bell: r.Bell,
		Detail: detailJSON{
			Kind: detailKinds[d.Kind], At: zoneText(d.At), Due: zoneText(d.Due), Project: d.Project,
			Priority: d.Priority, Category: d.Category, AutoFiled: d.AutoFiled,
		},
	}
	for _, ch := range d.Changes {
		out.Detail.Changes = append(out.Detail.Changes, changeJSON{Kind: changeKinds[ch.Kind], At: zoneText(ch.At), Value: ch.Value})
	}
	return out
}

var problemNames = map[providers.Kind]string{
	providers.KindOfficialQuota:     "quota",
	providers.KindOfficialSignedOut: "signed_out",
	providers.KindOfficialOff:       "off",
	providers.KindCancelled:         "cancelled",
	providers.KindTimeout:           "timeout",
	providers.KindUnreachable:       "unreachable",
	providers.KindRateLimited:       "rate_limited",
	providers.KindNotConfigured:     "off",
}

// problemOf is the kind of a failed turn, as Ask's JSON names it.
func problemOf(err error) string {
	if errors.Is(err, agent.ErrBusy) {
		return "busy"
	}
	if name, ok := problemNames[providers.Classify(err)]; ok {
		return name
	}
	return "other"
}

// assistantAgent is the phone's agent, built on first use: the official model only (the phone has
// no key of its own), the notes in the data folder, and the zone of the latest Ask for the clock.
func (c *Core) assistantAgent() *agent.Agent {
	a := &c.assist
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.agent == nil {
		sel := providers.Selector{Official: c.officialModel}
		a.agent = agent.New(agent.Config{
			Core: c.dom, Model: sel.Model, Memory: agent.NewMemory(filepath.Join(c.dataDir, assistantMemoryFile)),
			Now: func() time.Time {
				a.mu.Lock()
				loc := a.loc
				a.mu.Unlock()
				if loc != nil {
					return c.nowTime().In(loc)
				}
				return c.nowTime()
			},
		})
	}
	return a.agent
}

// Ask gives the official model what the person typed, in the place askPlace names (see
// askPlace), and blocks until it has answered: call it off the main thread, and only when
// ModelState says official. The reply is JSON (see replyJSON): a plain answer in text, the
// changes it made as receipts (undo them with Undo), or a question with its options to pick from.
// A model that could not answer is not an error: the reply names the problem and lists any
// receipts made before it stopped. StopAsk cancels the turn in flight; a second Ask while one runs
// answers problem busy.
func (c *Core) Ask(text, placeJSON string) (string, error) {
	var place askPlace
	if placeJSON != "" {
		if err := json.Unmarshal([]byte(placeJSON), &place); err != nil {
			return "", errors.New("could not read where the request came from")
		}
	}
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return "", errors.New("the core is closed")
	}
	ag := c.assistantAgent()
	ctx, cancel := context.WithCancel(context.Background())
	a := &c.assist
	a.mu.Lock()
	if a.cancel != nil { // a turn is running; the agent refuses a second one
		a.mu.Unlock()
		cancel()
		return marshal(replyJSON{Receipts: []receiptJSON{}, Problem: "busy", Error: agent.ErrBusy.Error()})
	}
	if loc := place.location(); loc != nil {
		a.loc = loc
	}
	a.cancel = cancel
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.cancel = nil
		a.mu.Unlock()
		cancel()
	}()

	res, err := ag.Ask(ctx, text, place.scope())
	c.refreshModelSoon() // the turn used one: the chip and the Settings row read the new count
	out := replyJSON{Text: res.Reply, Receipts: []receiptJSON{}}
	for _, r := range res.Receipts {
		out.Receipts = append(out.Receipts, receiptOf(r))
	}
	if q := res.Question; q != nil {
		out.Question = &questionJSON{Text: q.Text, Options: q.Options}
	}
	if len(res.Receipts) > 0 {
		c.notifyChange()
		c.nudgeSync()
	}
	if err != nil {
		out.Problem, out.Error = problemOf(err), providers.Friendly(err)
	}
	return marshal(out)
}

// StopAsk cancels the Ask in flight, if any. The Ask returns problem cancelled, with the receipts
// of the changes made before it stopped.
func (c *Core) StopAsk() {
	a := &c.assist
	a.mu.Lock()
	cancel := a.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Undo reverts the one change a receipt of Ask describes. A receipt the assistant no longer
// remembers (the app restarted) fails; the task is then changed by hand.
func (c *Core) Undo(receiptID int64) error {
	if err := c.assistantAgent().Undo(context.Background(), int(receiptID)); err != nil {
		return shortError(err)
	}
	c.notifyChange()
	c.nudgeSync()
	return nil
}

// Reset forgets the conversation, so the next Ask starts fresh. Undo of earlier receipts keeps
// working.
func (c *Core) Reset() { c.assistantAgent().Reset() }
