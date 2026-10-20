package domain_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// TestAddReplyIsTheUsersReportAndLeavesTheTask pins the domain half of the Reply contract: a
// report by "you" of kind "reply", no task change, and a hand-off still waiting for an agent.
func TestAddReplyIsTheUsersReportAndLeavesTheTask(t *testing.T) {
	c, clk := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "decide", Source: domain.SourceAgent, Status: domain.StatusBlocked})
	before := *task

	if _, err := c.AddReply(ctx, task.ID, "  I will decide tomorrow.  "); err != nil {
		t.Fatalf("AddReply: %v", err)
	}
	after, err := c.GetTask(ctx, task.ID)
	if err != nil || after == nil {
		t.Fatalf("GetTask: %v", err)
	}
	if !reflect.DeepEqual(before, *after) {
		t.Errorf("AddReply changed the task\nbefore: %+v\nafter:  %+v", before, *after)
	}
	reports := must[[]domain.Report](t)(c.ListReports(ctx, task.ID))
	if len(reports) != 1 {
		t.Fatalf("reports = %+v", reports)
	}
	reply := reports[0]
	if reply.Agent != domain.ReplyAgent || reply.Kind != domain.KindReply {
		t.Errorf("reply agent %q kind %q, want %q and %q", reply.Agent, reply.Kind, domain.ReplyAgent, domain.KindReply)
	}
	if reply.Summary != "I will decide tomorrow." {
		t.Errorf("reply summary = %q", reply.Summary)
	}
	if reply.Status != nil || reply.NextStep != nil || reply.Cwd != "" || reply.Links == nil || len(reply.Links) != 0 {
		t.Errorf("reply carries an agent's fields: %+v", reply)
	}

	// A reply is not the agent taking a handed-over task.
	handed := must[*domain.Task](t)(c.HandOff(ctx, task.ID))
	clk.advance(time.Minute)
	if _, err := c.AddReply(ctx, task.ID, "still waiting"); err != nil {
		t.Fatalf("AddReply on a handed-over task: %v", err)
	}
	afterHandOff, err := c.GetTask(ctx, task.ID)
	if err != nil || afterHandOff == nil {
		t.Fatalf("GetTask after hand-off: %v", err)
	}
	if afterHandOff.HandedTo == nil || *afterHandOff.HandedTo != domain.AnyAgent {
		t.Errorf("Reply cleared the hand-off: %+v", afterHandOff)
	}
	if !reflect.DeepEqual(handed.Status, afterHandOff.Status) || !reflect.DeepEqual(handed.HandedAt, afterHandOff.HandedAt) {
		t.Errorf("Reply changed the hand-off\nbefore: %+v\nafter:  %+v", handed, afterHandOff)
	}

	if _, err := c.AddReply(ctx, task.ID, "   "); err == nil {
		t.Error("AddReply accepted a blank text")
	}
	if _, err := c.AddReply(ctx, 9999, "hello"); !domain.IsNotFound(err) {
		t.Errorf("AddReply on a missing task = %v, want a not-found error", err)
	}
}
