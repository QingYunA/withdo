package views

import (
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/egoist/mygo/ui"
)

// SyncService is what Settings > Sync asks for its state and its actions. Real implements it over
// internal/syncclient; tests use a fake.
type SyncService interface {
	SyncSnapshot() data.SyncSnapshot
	SyncSignIn()
	SyncMerge()
	SyncUseAccount()
	SyncCancel()
	SyncSignOut()
	SyncNow()
	SyncRetry()
	SyncSetServer(server string) error
	SyncRefreshDevices()
}

// The sheet of the first sign-in that found tasks on both sides.
const (
	syncSheetWidth  float32 = 540
	syncSheetLeft   float32 = (theme.SettingsWidth - syncSheetWidth) / 2
	syncSheetTop    float32 = settingsBandHeight + 26
	syncHeroMaxBody float32 = 440
	syncBarCells    int     = 40
)

// settingsBandHeight is the toolbar: the title row over the pane icons and their hairline.
const settingsBandHeight float32 = SettingsTitleBarHeight + 56 + 12

// syncAttention is how long a failed sync may stay quiet in the main window before the sidebar
// says something.
const syncAttention = 15 * time.Minute

// SyncChoice is the option picked in the both-have-tasks sheet.
type SyncChoice int

const (
	SyncChoiceMerge SyncChoice = iota
	SyncChoiceAccount
)

// syncPane is Settings > Sync: the account and this Mac, the devices, and the server at the foot.
func (s *Settings) syncPane(c *ui.Context, t *theme.Tokens) {
	svc := s.app.Sync
	if svc == nil {
		group(c, t, "sync-none", func() {
			settingsRow(c, t, "none", s.say.SettingsSyncUnavailable(), "", true, nil)
		})
		return
	}
	snap := svc.SyncSnapshot()
	if !s.syncAsked {
		s.syncAsked = true
		s.syncLastSynced = snap.LastSynced
		svc.SyncRefreshDevices()
	} else if !snap.LastSynced.IsZero() && !snap.LastSynced.Equal(s.syncLastSynced) {
		s.syncLastSynced = snap.LastSynced
		svc.SyncRefreshDevices()
	}
	switch snap.Phase {
	case data.SyncSigningIn, data.SyncUploading:
		c.After(time.Second)
	case data.SyncSynced, data.SyncFailed, data.SyncOffline:
		c.After(30 * time.Second)
	}
	switch snap.Phase {
	case data.SyncSignedOut:
		s.syncSignedOut(c, t, svc)
	case data.SyncSigningIn:
		s.syncSigningIn(c, t)
	case data.SyncBothHaveTasks:
		group(c, t, "sync-account", func() { s.syncAccountRow(c, t, svc, snap, true) })
	case data.SyncUploading, data.SyncSynced:
		s.syncAccountGroup(c, t, svc, snap, snap.Phase == data.SyncUploading)
		if snap.Phase == data.SyncUploading {
			theme.Paragraph(c, theme.Meta, t.TextTertiary, s.say.SettingsSyncUploadNote(), 0).Margin(12, 4, 0)
		}
		s.syncDevices(c, t, snap)
	default: // failed or offline
		s.syncAccountGroup(c, t, svc, snap, false)
	}
	s.syncFooter(c, t, svc, snap)
	if snap.Conflict != nil {
		if !s.syncHadConflict {
			s.syncChoice, s.syncHadConflict = SyncChoiceMerge, true
		}
	} else {
		s.syncHadConflict = false
	}
}

// syncSignedOut is the centered card of a Mac that is not signed in.
func (s *Settings) syncSignedOut(c *ui.Context, t *theme.Tokens, svc SyncService) {
	group(c, t, "sync-hero", func() {
		ui.Column(c).FillWidth().AlignItems(ui.Center).Padding(28, 40, 24).Children(func() {
			ui.Icon(c, theme.IconCloud).Size(theme.IconLarge, theme.IconLarge).TextColor(t.Accent)
			hero := theme.Input
			hero.Weight = 600
			theme.Paragraph(c, hero, t.Text, s.say.SettingsSyncSignedOutTitle(), 0).TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(12, 0, 0)
			theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncSignedOutBody(), 0).
				TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(6, 0, 0)
			if settingsButton(c, t, "sync-signin", s.say.SettingsSyncSignIn(), true).Margin(16, 0, 0).Clicked() {
				svc.SyncSignIn()
			}
			theme.Paragraph(c, theme.Meta, t.TextTertiary, s.say.SettingsSyncSignInNote(), 0).
				TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(10, 0, 0)
		})
	})
}

// syncSigningIn is the card while the browser owns the sign-in.
func (s *Settings) syncSigningIn(c *ui.Context, t *theme.Tokens) {
	group(c, t, "sync-hero", func() {
		ui.Column(c).FillWidth().AlignItems(ui.Center).Padding(28, 40, 24).Children(func() {
			ui.Icon(c, theme.IconCloud).Size(theme.IconLarge, theme.IconLarge).TextColor(t.Accent)
			hero := theme.Input
			hero.Weight = 600
			theme.Paragraph(c, hero, t.Text, s.say.SettingsSyncSignedOutTitle(), 0).TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(12, 0, 0)
			theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncSigningInBody(), 0).
				TextAlign(ui.Center).MaxWidth(syncHeroMaxBody).Margin(6, 0, 0)
			settingsButton(c, t, "sync-signing", s.say.SettingsSyncSigningIn(), true).Disabled(true).Margin(16, 0, 0)
		})
	})
}

// syncAccountGroup is the account row over this Mac's row.
func (s *Settings) syncAccountGroup(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot, uploading bool) {
	group(c, t, "sync-account", func() {
		s.syncAccountRow(c, t, svc, snap, true)
		if uploading {
			s.syncUploadingRow(c, t, snap)
		} else {
			s.syncThisMacRow(c, t, svc, snap)
		}
	})
}

func (s *Settings) syncAccountRow(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot, first bool) {
	syncRow(c, t, "account", first, func() {
		theme.Text(c, theme.Body, t.Text, s.syncAccountName(snap)).Ellipsis("…")
		theme.Text(c, theme.Meta, t.TextSecondary, s.syncAccountLine(snap))
	}, func() {
		if settingsLink(c, t, "sync-signout", s.say.SettingsSyncSignOut(), s.say.SettingsSyncSignOut()).Clicked() {
			svc.SyncSignOut()
		}
	})
}

func (s *Settings) syncThisMacRow(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	failed := snap.Phase == data.SyncFailed
	offline := snap.Phase == data.SyncOffline
	syncRow(c, t, "this-mac", false, func() {
		theme.Text(c, theme.Body, t.Text, s.say.SettingsSyncThisMac())
		switch {
		case failed:
			ui.Row(c).Gap(0).AlignItems(ui.Center).Children(func() {
				theme.Text(c, theme.Meta, t.Blocked(), s.say.SettingsSyncFailed(s.syncFailureText(snap))).Shrink(1).MinWidth(0).Ellipsis("…")
				theme.Text(c, theme.Meta, t.TextSecondary, " · "+s.syncRetryLine(snap.RetryAt)).Shrink(0)
			})
			theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncWaiting(itoa(snap.Pending)), 2)
		case offline:
			theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncOffline(itoa(snap.Pending)), 2)
		default:
			at := snap.LastSynced
			if at.IsZero() {
				at = s.app.Now()
			}
			ago := s.clock.Age(at, s.app.Now())
			if s.app.Now().Sub(at) < time.Minute {
				ago = s.say.SettingsSyncJustNow()
			}
			ui.Row(c).Gap(0).AlignItems(ui.Center).Children(func() {
				theme.Text(c, theme.Meta, t.Status(data.Done), s.say.SettingsSyncSynced())
				theme.Text(c, theme.Meta, t.TextSecondary, " · "+ago)
			})
		}
	}, func() {
		switch {
		case failed:
			if settingsButton(c, t, "sync-retry", s.say.SettingsSyncRetry(), false).Clicked() {
				svc.SyncRetry()
			}
		case offline:
			// Offline has no button: the runner tries by itself.
		default:
			if settingsButton(c, t, "sync-now", s.say.SettingsSyncNow(), false).Clicked() {
				svc.SyncNow()
			}
		}
	})
}

// syncUploadingRow is the first device seeding the account: the count and the pixel bar.
func (s *Settings) syncUploadingRow(c *ui.Context, t *theme.Tokens, snap data.SyncSnapshot) {
	syncRow(c, t, "uploading", false, func() {
		theme.Text(c, theme.Body, t.Text, s.say.SettingsSyncUploading())
		theme.Paragraph(c, theme.Meta, t.TextSecondary, s.say.SettingsSyncUploadProgress(groupThousands(snap.SeedDone), groupThousands(snap.SeedTotal)), 2)
		syncBar(c, t, "sync-upload", snap.SeedDone, snap.SeedTotal)
	}, nil)
}

// syncDevices is the group under the account: this Mac first, then every other device.
func (s *Settings) syncDevices(c *ui.Context, t *theme.Tokens, snap data.SyncSnapshot) {
	if len(snap.Devices) == 0 {
		return
	}
	groupLabel(c, t, s.say.SettingsSyncDevices())
	devices := append([]data.SyncDevice(nil), snap.Devices...)
	sort.SliceStable(devices, func(i, j int) bool { return devices[i].This && !devices[j].This })
	group(c, t, "sync-devices", func() {
		for i, d := range devices {
			s.syncDeviceRow(c, t, d, i == 0)
		}
	})
}

func (s *Settings) syncDeviceRow(c *ui.Context, t *theme.Tokens, d data.SyncDevice, first bool) {
	if !first {
		ui.Box(c.Key("sync-device-rule-"+d.ID)).Height(1).Shrink(0).Margin(0, 14).Background(t.Border)
	}
	ui.Row(c.Key("sync-device-"+d.ID)).MinHeight(theme.SettingsRowMin).Shrink(0).Padding(8, 14).Gap(12).AlignItems(ui.Center).Children(func() {
		icon := theme.IconLaptop
		if phoneName(d.Name) {
			icon = theme.IconPhone
		}
		ui.Icon(c, icon).Size(theme.IconRow, theme.IconRow).TextColor(t.TextTertiary).Shrink(0)
		ui.Row(c).Grow(1).MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			theme.Text(c, theme.Body, t.Text, d.Name).Shrink(1).MinWidth(0).Ellipsis("…")
			if d.This {
				syncBadge(c, t, s.say.SettingsSyncThisMacBadge())
			}
		})
		age := s.say.SettingsSyncNever()
		if !d.LastSeen.IsZero() {
			age = s.clock.Age(d.LastSeen, s.app.Now())
		}
		theme.Text(c, theme.Meta, t.TextSecondary, age).Shrink(0)
	})
}

// syncFooter is the server line: the host, and Change, which edits it in place.
func (s *Settings) syncFooter(c *ui.Context, t *theme.Tokens, svc SyncService, snap data.SyncSnapshot) {
	ui.Row(c.Key("sync-footer")).Margin(14, 4, 0).Shrink(0).Gap(6).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Meta, t.TextTertiary, s.say.SettingsSyncServer())
		if s.syncEditingServer {
			fieldWell(c, t, "sync-server", 330, func() {
				var field ui.Element
				field = ui.TextInputBase(c.Key("sync-server"), &s.syncServerDraft)
				theme.Body.Apply(field).TextColor(t.Text).Label(s.say.SettingsSyncServerAddress()).AutoFocus().
					OnTextKey(func(k ui.TextKey) bool {
						switch k.Key {
						case ui.KeyEnter:
							server := strings.TrimSpace(s.syncServerDraft)
							if err := svc.SyncSetServer(server); err != nil {
								s.syncServerErr = true
								return true
							}
							s.syncEditingServer, s.syncServerErr = false, false
							return true
						case ui.KeyEscape:
							s.syncEditingServer, s.syncServerErr = false, false
							return true
						}
						return false
					})
			})
			if s.syncServerErr {
				theme.Text(c, theme.Meta, t.Blocked(), s.say.SettingsSyncServerInvalid())
			}
			return
		}
		theme.Text(c, theme.Mono, t.TextTertiary, serverHost(snap.Server)).Shrink(1).MinWidth(0).Ellipsis("…")
		theme.Text(c, theme.Meta, t.TextTertiary, "·")
		if settingsLink(c, t, "sync-change", s.say.SettingsSyncChange(), s.say.SettingsSyncServerAddress()).Clicked() {
			s.syncEditingServer = true
			s.syncServerDraft = snap.Server
			s.syncServerErr = false
		}
	})
}

// syncSheet asks what wins when this Mac and the account both have tasks.
func (s *Settings) syncSheet(c *ui.Context, t *theme.Tokens, snap data.SyncSnapshot) {
	if s.pane != PaneSync || s.app.Sync == nil || snap.Conflict == nil {
		return
	}
	cancelled := false
	ui.Overlay(c.Key("sync-sheet"), func() {
		back := ui.Box(c.Key("sync-scrim")).Absolute().Left(0).Top(settingsBandHeight).Right(0).Bottom(0).
			Background(ui.RGBA(0, 0, 0, 0.18)).Modal()
		if back.Clicked() || back.OverlayShortcut(0, ui.KeyEscape) {
			cancelled = true
		}
		panel := ui.Column(c.Key("sync-sheet-panel")).Absolute().Left(syncSheetLeft).Top(syncSheetTop).
			Width(syncSheetWidth).Padding(20, 22).Radius(theme.RadiusPanel).Background(t.SettingsGroup()).
			Border(1, t.Border).Shadow(0, 18, 40, 0, t.Shadow).Role(ui.RoleDialog).Modal()
		panel.Children(func() {
			title := theme.Input
			title.Weight = 600
			theme.Text(c, title, t.Text, s.say.SettingsSyncBothTitle())
			theme.Paragraph(c, theme.Meta, t.TextSecondary,
				s.say.SettingsSyncBothBody(itoa(snap.Conflict.AccountDevices), itoa(snap.Conflict.LocalTasks), itoa(snap.Conflict.LocalProjects)), 0).
				Margin(6, 0, 0)
			if syncChoiceCard(c, t, "sync-merge", s.say.SettingsSyncMerge(), s.say.SettingsSyncMergeBody(),
				s.syncChoice == SyncChoiceMerge, s.say.SettingsSyncRecommended()) {
				s.syncChoice = SyncChoiceMerge
			}
			if syncChoiceCard(c, t, "sync-account-choice", s.say.SettingsSyncUseAccount(), s.say.SettingsSyncUseAccountBody(),
				s.syncChoice == SyncChoiceAccount, "") {
				s.syncChoice = SyncChoiceAccount
			}
			ui.Row(c).Justify(ui.End).Gap(8).Margin(18, 0, 0).Children(func() {
				if settingsButton(c, t, "sync-cancel", s.say.SettingsSyncCancel(), false).Clicked() {
					cancelled = true
				}
				if settingsButton(c, t, "sync-continue", s.say.SettingsSyncContinue(), true).Clicked() {
					if s.syncChoice == SyncChoiceAccount {
						s.app.Sync.SyncUseAccount()
					} else {
						s.app.Sync.SyncMerge()
					}
				}
			})
		})
	})
	if cancelled {
		s.syncHadConflict = false
		s.app.Sync.SyncCancel()
	}
}

// syncChoiceCard is one option of the both-have-tasks sheet: a pixel radio, a title and the one
// sentence that says what it does.
func syncChoiceCard(c *ui.Context, t *theme.Tokens, key, title, body string, selected bool, recommended string) bool {
	card := ui.ButtonBase(c.Key(key)).FillWidth().Row().Padding(12, 14).Radius(10).Gap(12).AlignItems(ui.Start).Cursor(ui.CursorPointer)
	if selected {
		card.Border(2, t.Accent)
	} else {
		card.Border(1, t.Border)
		if card.Hovered() {
			card.Background(t.BgHover)
		}
	}
	card.Margin(12, 0, 0)
	card.Children(func() {
		ink := t.TextTertiary
		if selected {
			ink = t.Accent
		}
		ui.Box(c).Size(16, 16).Center().Shrink(0).Margin(3, 0, 0).Children(func() {
			theme.RadioGlyph(c, ink, selected)
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				theme.Text(c, theme.Body, t.Text, title)
				if recommended != "" {
					ui.Row(c).Padding(0, 5).Height(16).Radius(5).Background(t.Accent.Alpha(0.12)).AlignItems(ui.Center).Children(func() {
						theme.Text(c, theme.Badge, t.Accent, recommended)
					})
				}
			})
			theme.Paragraph(c, theme.Meta, t.TextSecondary, body, 0)
		})
	})
	return card.Clicked()
}

// syncRow is a row of the account group whose left side is built by hand, for the two lines of a
// failure or the bar of an upload.
func syncRow(c *ui.Context, t *theme.Tokens, key string, first bool, body, control func()) {
	if !first {
		ui.Box(c.Key("sync-rule-"+key)).Height(1).Shrink(0).Margin(0, 14).Background(t.Border)
	}
	ui.Row(c.Key("sync-row-"+key)).MinHeight(theme.SettingsRowMin).Shrink(0).Padding(8, 14).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(body)
		if control != nil {
			control()
		}
	})
}

// syncBar is the 40 cell pixel bar of the first upload: accent for the done share, 8% black for
// the rest.
func syncBar(c *ui.Context, t *theme.Tokens, key string, done, total int) {
	filled := syncBarFilled(done, total)
	ui.Row(c.Key(key)).Gap(2).Margin(10, 0, 0).Shrink(0).Children(func() {
		for i := 0; i < syncBarCells; i++ {
			cell := ui.Box(c.Key(key + "-" + itoa(i))).Width(9).Height(8).Shrink(0).Radius(1)
			if i < filled {
				cell.Background(t.Accent)
			} else {
				cell.Background(t.Text.Alpha(0.08))
			}
		}
	})
}

// syncBarFilled is how many of the 40 cells are lit: the rounded done share, and at least one once
// anything is done.
func syncBarFilled(done, total int) int {
	if done <= 0 || total <= 0 {
		return 0
	}
	n := (done*syncBarCells + total/2) / total
	if n < 1 {
		n = 1
	}
	if n > syncBarCells {
		n = syncBarCells
	}
	return n
}

// syncBadge is the outlined "This Mac" badge of a device row.
func syncBadge(c *ui.Context, t *theme.Tokens, text string) {
	ui.Row(c).Padding(0, 5).Height(16).Shrink(0).Radius(5).Border(1, t.Border).AlignItems(ui.Center).Children(func() {
		theme.Text(c, theme.Badge, t.TextSecondary, text)
	})
}

// syncAccountName is the account's display name, else its email.
func (s *Settings) syncAccountName(snap data.SyncSnapshot) string {
	if snap.Account != "" {
		return snap.Account
	}
	if snap.AccountEmail != "" {
		return snap.AccountEmail
	}
	return s.say.SettingsSyncAccount()
}

func (s *Settings) syncAccountLine(snap data.SyncSnapshot) string {
	switch len(snap.Devices) {
	case 0:
		return s.say.SettingsSyncSignedIn()
	case 1:
		return s.say.SettingsSyncSignedInOneDevice()
	}
	return s.say.SettingsSyncSignedInDevices(itoa(len(snap.Devices)))
}

// syncFailureText words the reason of a failed sync in the window's language. An error of a kind
// the pane does not know reads as the server not answering; the technical text stays in the
// snapshot for the logs.
func (s *Settings) syncFailureText(snap data.SyncSnapshot) string {
	switch snap.Failure {
	case data.SyncFailureAuth:
		return s.say.SettingsSyncFailedAuth()
	}
	return s.say.SettingsSyncFailedNoAnswer()
}

// syncRetryLine says when a failed sync tries again.
func (s *Settings) syncRetryLine(at time.Time) string {
	if at.IsZero() {
		return s.say.SettingsSyncRetryingSoon()
	}
	d := at.Sub(s.app.Now())
	switch {
	case d < time.Minute:
		return s.say.SettingsSyncRetryingSoon()
	case d < time.Hour:
		return s.say.SettingsSyncRetryingMinutes(itoa(int(d / time.Minute)))
	}
	return s.say.SettingsSyncRetryingHours(itoa(int(d / time.Hour)))
}

// SyncNeedsAttention says whether the sidebar shows the red cloud: a failed sync older than 15
// minutes, or a linked Mac that signed out.
func (a *App) SyncNeedsAttention() bool {
	if a.Sync == nil {
		return false
	}
	snap := a.Sync.SyncSnapshot()
	if !snap.Linked {
		return false
	}
	if snap.Phase == data.SyncSignedOut {
		return true
	}
	return snap.Phase == data.SyncFailed && !snap.FailedSince.IsZero() && a.Now().Sub(snap.FailedSince) > syncAttention
}

// serverHost is what the footer shows: the host of the server URL, without the scheme.
func serverHost(server string) string {
	if u, err := url.Parse(server); err == nil && u.Host != "" {
		return u.Host
	}
	return strings.TrimPrefix(strings.TrimPrefix(server, "https://"), "http://")
}

// phoneName guesses a phone from the name the server has for a device: the server does not say
// what kind of device it is.
func phoneName(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "phone") || strings.Contains(n, "iphone") ||
		strings.Contains(n, "android") || strings.Contains(n, "pixel")
}

// groupThousands puts commas in a count: 3512 -> 3,512.
func groupThousands(n int) string {
	if n < 0 {
		return "-" + groupThousands(-n)
	}
	raw := itoa(n)
	var b strings.Builder
	for i, r := range raw {
		if i > 0 && (len(raw)-i)%3 == 0 && unicode.IsDigit(r) {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}
