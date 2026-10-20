package core

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// The two answers of SignIn. "linked" says this phone is a device of the account; "choose" says
// both the phone and the account have tasks, so the user must pick LinkMerge or LinkReplace.
const (
	signInLinkedJSON = `{"state":"linked"}`
	signInChooseJSON = `{"state":"choose"}`
)

// Account returns who this phone signed in as, from sync.json:
//
//	{"name":"Ada Lovelace","email":"ada@example.com"}
//
// It is {} when the phone has never signed in or is signed out (its refresh token is gone).
func (c *Core) Account() (string, error) {
	cfg, found, err := c.loadConfig()
	if err != nil {
		return "", err
	}
	if !found || cfg.Account == nil {
		return "{}", nil
	}
	if cfg.Token == "" {
		if c.secrets == nil {
			return "{}", nil
		}
		if _, err := (secretAdapter{c.secrets}).Get(syncclient.RefreshTokenKey); err != nil {
			if errors.Is(err, syncclient.ErrSecretNotFound) {
				return "{}", nil
			}
			return "", errors.New("could not read the account")
		}
	}
	return marshal(cfg.Account)
}

// deviceJSON is one row of Devices: syncproto.Device's fields and whether this is this phone.
type deviceJSON struct {
	syncproto.Device
	This bool `json:"this"`
}

// Devices returns the account's devices from the server, the one that synced most recently first,
// with "this":true on this phone:
//
//	[{"id":"...","name":"Pixel","last_seen_at":"2026-10-20T09:00:00Z","this":true},
//	 {"id":"...","name":"Mac","last_seen_at":null,"this":false}]
//
// last_seen_at is null before a device's first sync. It reads the server, so it needs a connection.
func (c *Core) Devices() (string, error) {
	client, err := c.client()
	if err != nil {
		return "", err
	}
	devices, err := client.Devices(context.Background())
	if err != nil {
		return "", syncError(err)
	}
	deviceID, _, err := c.dom.SyncState(context.Background(), "device_id")
	if err != nil {
		return "", errors.New("could not read the link")
	}
	return encodeDevices(devices, deviceID)
}

// encodeDevices is Devices' encoder, split out so the golden test can pin its shape.
func encodeDevices(devices []syncclient.Device, thisID string) (string, error) {
	out := make([]deviceJSON, 0, len(devices))
	for _, device := range devices {
		out = append(out, deviceJSON{Device: device, This: device.ID == thisID})
	}
	return marshal(out)
}

// reminderJSON is one row of ClaimDueReminders. Project is null for an Inbox task.
type reminderJSON struct {
	TaskID  int64   `json:"task_id"`
	Title   string  `json:"title"`
	Project *string `json:"project"`
}

// ClaimDueReminders claims the reminders due now and returns them as JSON:
//
//	[{"task_id":7,"title":"Pay rent","project":"Home"},
//	 {"task_id":9,"title":"Stretch","project":null}]
//
// Each reminder is returned once: claiming marks it reminded in the database, and the sync client
// uploads that mark like any other change. A phone with none returns [].
func (c *Core) ClaimDueReminders() (string, error) {
	due, err := c.dom.ClaimDueReminders(context.Background(), c.nowTime())
	if err != nil {
		return "", errors.New("could not check reminders")
	}
	return encodeReminders(due)
}

// encodeReminders is ClaimDueReminders' encoder, split out so the golden test can pin its shape.
func encodeReminders(due []domain.Reminder) (string, error) {
	out := make([]reminderJSON, 0, len(due))
	for _, reminder := range due {
		out = append(out, reminderJSON{
			TaskID: reminder.TaskID, Title: reminder.Title, Project: reminder.ProjectName,
		})
	}
	return marshal(out)
}

// NextReminderAt returns the earliest remind_at of an open task that has not been reminded yet, in
// Unix milliseconds, or 0 when there is none. The phone sets an alarm for it; ClaimDueReminders
// then collects the reminder when the alarm fires.
func (c *Core) NextReminderAt() (int64, error) {
	var stored sql.NullString
	err := c.db.QueryRowContext(context.Background(), `SELECT MIN(remind_at) FROM tasks
		WHERE remind_at IS NOT NULL AND reminded_at IS NULL
		AND status NOT IN ('done','cancelled')`).Scan(&stored)
	if err != nil {
		return 0, errors.New("could not check reminders")
	}
	if !stored.Valid || strings.TrimSpace(stored.String) == "" {
		return 0, nil
	}
	at, err := time.Parse(time.RFC3339Nano, stored.String)
	if err != nil {
		return 0, errors.New("could not check reminders")
	}
	return at.UnixMilli(), nil
}

// LinkMerge joins the account without losing this phone's tasks, after SignIn answered "choose".
// It is Client.LinkMerge: both sides keep their tasks and the account's rows win where the same
// project or category exists.
func (c *Core) LinkMerge(deviceName string) error {
	client, err := c.newClient(false)
	if err != nil {
		return err
	}
	if _, err := client.LinkMerge(context.Background(), strings.TrimSpace(deviceName)); err != nil {
		return linkError(err)
	}
	return nil
}

// LinkReplace backs this phone up and joins the account with the account's tasks only, after
// SignIn answered "choose". It returns the backup file's path. The backup is taken before anything
// local is deleted.
func (c *Core) LinkReplace(deviceName string) (string, error) {
	client, err := c.newClient(false)
	if err != nil {
		return "", err
	}
	backup, err := client.LinkReplace(context.Background(), strings.TrimSpace(deviceName))
	if err != nil {
		return backup, linkError(err)
	}
	return backup, nil
}

// linkError turns a merge or replace failure into a sentence fit to show the user. It never
// carries a path, a SQL fragment or a token.
func linkError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, syncclient.ErrSeeding):
		return errors.New("another device is still uploading this workspace; try again later")
	case errors.Is(err, syncclient.ErrLocalData):
		return errors.New("this phone already has tasks of its own")
	case errors.Is(err, syncclient.ErrNotLinked):
		return errors.New("this phone is not linked")
	case syncclient.IsSignedOut(err), syncclient.IsUnauthorized(err):
		return errors.New("sign in again")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return errors.New("the link was cancelled")
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "already linked"):
		return errors.New("this phone is already linked")
	case strings.Contains(message, "workspace is empty"):
		return errors.New("the account has no tasks yet")
	case strings.Contains(message, "device name is required"):
		return errors.New("a device name is required")
	}
	return syncError(err)
}
