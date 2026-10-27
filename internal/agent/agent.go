package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const (
	// DefaultIdleReset is how long a conversation is remembered without a new message.
	DefaultIdleReset = 10 * time.Minute
	maxIterations    = 8
	maxHistory       = 60 // messages kept for follow-ups
)

// ErrBusy is returned by Ask while another request is still running.
var ErrBusy = errors.New("the assistant is still working on the last request")

// ModelFunc returns the chat model to use right now. It is called for every turn, so a change of
// provider in Settings applies to the next request without a restart.
type ModelFunc func(ctx context.Context) (model.ToolCallingChatModel, error)

// Config wires an Agent. Now's location is the user's zone.
type Config struct {
	Core      *domain.Core
	Model     ModelFunc
	Now       func() time.Time // time.Now in the user's zone by default
	IdleReset time.Duration    // DefaultIdleReset when zero
	// Memory is the user's notes, read into every turn's prompt; nil gives the assistant no memory.
	Memory *Memory
}

// Agent is the built-in assistant. It remembers the current conversation in memory only.
type Agent struct {
	cfg     Config
	journal *journal

	mu      sync.Mutex
	history []*schema.Message
	last    time.Time
	busy    bool
}

// New returns an Agent with an empty conversation.
func New(cfg Config) *Agent {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.IdleReset == 0 {
		cfg.IdleReset = DefaultIdleReset
	}
	return &Agent{cfg: cfg, journal: newJournal()}
}

// Reset forgets the conversation. Undo of earlier receipts keeps working.
func (a *Agent) Reset() {
	a.mu.Lock()
	a.history, a.last = nil, time.Time{}
	a.mu.Unlock()
}

// Undo reverts the one change a receipt describes.
func (a *Agent) Undo(ctx context.Context, receiptID int) error { return a.journal.run(ctx, receiptID) }

// Image is a picture sent with the user's text. It lives in memory only, as part of the
// conversation.
type Image struct {
	MIME string // image/jpeg, image/png, image/gif or image/webp
	Data []byte
}

// userMessage is what the user said: plain text, or the text followed by its pictures.
func userMessage(text string, images []Image) *schema.Message {
	if len(images) == 0 {
		return schema.UserMessage(text)
	}
	parts := make([]schema.MessageInputPart, 0, len(images)+1)
	if strings.TrimSpace(text) != "" {
		parts = append(parts, schema.MessageInputPart{Type: schema.ChatMessagePartTypeText, Text: text})
	}
	for _, im := range images {
		b64 := base64.StdEncoding.EncodeToString(im.Data)
		parts = append(parts, schema.MessageInputPart{Type: schema.ChatMessagePartTypeImageURL, Image: &schema.MessageInputImage{
			MessagePartCommon: schema.MessagePartCommon{Base64Data: &b64, MIMEType: im.MIME},
		}})
	}
	return &schema.Message{Role: schema.User, UserInputMultiContent: parts}
}

// Ask runs one turn: the user's text, as a continuation of the current conversation, typed in
// the place scope names. It blocks until the model has answered, so call it off the UI thread;
// cancelling ctx stops it. On error the Result still lists the changes made before the turn
// stopped.
func (a *Agent) Ask(ctx context.Context, text string, scope Scope, images ...Image) (Result, error) {
	msgs, err := a.begin(userMessage(text, images))
	if err != nil {
		return Result{}, err
	}
	defer a.finish()
	tb, runner, err := a.prepare(ctx, scope, len(images))
	if err != nil {
		return Result{}, err
	}
	produced, runErr := collect(runner.Run(ctx, msgs))
	if runErr == nil && scope.Kind == ScopeTask {
		tb.keep(scope.TaskID, nil) // a request from the inspector's composer belongs to that task
	}
	res := Result{Receipts: tb.takeReceipts(), Question: tb.asked(), Images: tb.kept()}
	if runErr != nil {
		return res, fmt.Errorf("agent: %w", runErr)
	}
	if res.Question == nil {
		res.Reply = finalReply(produced)
	}
	a.remember(append(msgs, produced...))
	return res, nil
}

// begin starts a turn: it refuses a second one and forgets a stale conversation.
func (a *Agent) begin(user *schema.Message) ([]*schema.Message, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy {
		return nil, ErrBusy
	}
	if !a.last.IsZero() && a.cfg.Now().Sub(a.last) > a.cfg.IdleReset {
		a.history = nil
	}
	a.busy = true
	return append(append([]*schema.Message(nil), a.history...), user), nil
}

func (a *Agent) finish() {
	a.mu.Lock()
	a.busy = false
	a.mu.Unlock()
}

func (a *Agent) remember(msgs []*schema.Message) {
	if len(msgs) > maxHistory {
		msgs = msgs[len(msgs)-maxHistory:]
		for len(msgs) > 0 && msgs[0].Role != schema.User { // start on a user message, not mid tool call
			msgs = msgs[1:]
		}
	}
	a.mu.Lock()
	a.history, a.last = msgs, a.cfg.Now()
	a.mu.Unlock()
}

// prepare builds this turn's tools and the Eino runner. The agent is cheap, so it is rebuilt per
// turn with a fresh prompt: the clock and the project list change between turns.
func (a *Agent) prepare(ctx context.Context, scope Scope, images int) (*toolbox, *adk.Runner, error) {
	m, err := a.cfg.Model(ctx)
	if err != nil {
		return nil, nil, err
	}
	projects, err := a.cfg.Core.ListProjects(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list projects: %w", err)
	}
	categories, err := a.cfg.Core.ListCategories(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list categories: %w", err)
	}
	focus, err := a.focusOf(ctx, scope)
	if err != nil {
		return nil, nil, err
	}
	tb := &toolbox{core: a.cfg.Core, now: a.cfg.Now, journal: a.journal, projects: projects, categories: categories, images: images, memory: a.cfg.Memory}
	if p, ok := activeProject(projects, scope.ProjectID); ok && scope.Kind == ScopeProject {
		tb.defaultProject = &p.ID
	}
	if scope.Kind == ScopeNow && slices.ContainsFunc(categories, func(c domain.Category) bool { return c.ID == scope.CategoryID }) {
		tb.defaultCategory = &scope.CategoryID
	}
	tools, direct, err := tb.tools()
	if err != nil {
		return nil, nil, err
	}
	prompt := BuildPrompt(a.cfg.Now(), projects, categories, scope, focus) + imageRules(images) + a.memoryPrompt()
	ag, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: "withdo", Description: "Turns requests into task changes.",
		Instruction: prompt, Model: m, MaxIterations: maxIterations,
		ToolsConfig: adk.ToolsConfig{ToolsNodeConfig: toolsNode(tools), ReturnDirectly: direct},
		// The prompt has literal braces nowhere today, but it must never be treated as a template.
		GenModelInput: func(_ context.Context, instruction string, in *adk.AgentInput) ([]*schema.Message, error) {
			return append([]*schema.Message{schema.SystemMessage(instruction)}, in.Messages...), nil
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("build agent: %w", err)
	}
	return tb, adk.NewRunner(ctx, adk.RunnerConfig{Agent: ag}), nil
}

// memoryPrompt is the rules for the memory tools and the notes, read afresh so a hand edit counts.
func (a *Agent) memoryPrompt() string {
	if a.cfg.Memory == nil {
		return ""
	}
	return memoryRules + userNotes(a.cfg.Memory.Text())
}

// focusOf is the task of a task scope. A task that was deleted since the inspector opened is an
// error the user can read, not a request about nothing.
func (a *Agent) focusOf(ctx context.Context, scope Scope) (*domain.Task, error) {
	if scope.Kind != ScopeTask {
		return nil, nil
	}
	t, err := a.cfg.Core.GetTask(ctx, scope.TaskID)
	if err != nil && !domain.IsNotFound(err) {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if t == nil || err != nil {
		return nil, errors.New("that task no longer exists")
	}
	return t, nil
}

// collect drains the event stream into the messages the turn produced (assistant tool calls,
// tool results, the final answer), stopping at the first error.
func collect(iter *adk.AsyncIterator[*adk.AgentEvent]) ([]*schema.Message, error) {
	var out []*schema.Message
	for {
		ev, ok := iter.Next()
		if !ok {
			return out, nil
		}
		if ev.Err != nil {
			drain(iter)
			return out, ev.Err
		}
		if mo := messageOf(ev); mo != nil {
			out = append(out, mo)
		}
	}
}

func drain(iter *adk.AsyncIterator[*adk.AgentEvent]) {
	for {
		if _, ok := iter.Next(); !ok {
			return
		}
	}
}

func messageOf(ev *adk.AgentEvent) *schema.Message {
	if ev.Output == nil || ev.Output.MessageOutput == nil || ev.Output.MessageOutput.IsStreaming {
		return nil
	}
	return ev.Output.MessageOutput.Message
}

// finalReply is the last assistant text that is not a tool call.
func finalReply(msgs []*schema.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if m := msgs[i]; m.Role == schema.Assistant && len(m.ToolCalls) == 0 {
			return strings.TrimSpace(m.Content)
		}
	}
	return ""
}
