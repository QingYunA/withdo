package views

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
	"github.com/QingYunA/adhd-todo/internal/speech"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/egoist/mygo/ui"
)

// Voice input (m12, decision 4 of "Input first"): the microphone button at the right of the page's
// composer records up to 60 s, the app uploads the recording to the sync server, which returns the
// text, and the text is sent as if it had been typed. No words show while speaking, so the line above
// the box shows the voice's level as a pixel waveform, then the busy indicator while the text comes.

type voicePhase int

const (
	voiceIdle voicePhase = iota
	voiceListening
	voiceTranscribing
)

// voiceProblem is why the last try ended without text; the line above the box words it.
type voiceProblem int

const (
	voiceNoProblem voiceProblem = iota
	voiceNeedSignIn
	voiceMicOff
	voiceQuotaHit
	voiceNotSetUp
	voiceCouldNot
)

const (
	// voiceProblemFor is how long the line about a failure stays.
	voiceProblemFor = 6 * time.Second
	// voiceButtonSize is the button; the picture in it is 24 pt (2 pt a pixel of the 12 x 12 grid).
	voiceButtonSize float32 = 32
	voiceMicSize    float32 = 24
	// voiceSettingsURL opens System Settings on Privacy & Security > Microphone.
	voiceSettingsURL = "x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone"
)

// voiceState is the microphone's state. peak is written from the audio thread; everything else is
// touched on the UI thread only.
type voiceState struct {
	phase     voicePhase
	owner     surface
	startedAt time.Time
	bars      [theme.WaveBars]float32 // the last 12 beats' levels, oldest first
	beat      int64                   // the 100 ms beat the bars were last pushed on
	peak      atomic.Uint32           // loudest level since the last beat (math.Float32bits)
	problem   voiceProblem
	problemAt time.Time
	cancel    context.CancelFunc // the upload in flight
	gen       int                // counts the tries, so an answer to a cancelled one is dropped
}

// noteLevel keeps the loudest level of the beat. It runs on the audio thread.
func (v *voiceState) noteLevel(level float32) {
	bits := math.Float32bits(level)
	for {
		old := v.peak.Load()
		if level <= math.Float32frombits(old) || v.peak.CompareAndSwap(old, bits) {
			return
		}
	}
}

// voiceReady says the composer shows a microphone: a recorder is wired in and works here.
func (m *Main) voiceReady() bool {
	if m.app.Recorder == nil || m.app.Act == nil {
		return false
	}
	ok, _ := m.app.Recorder.Available()
	return ok
}

// voiceSignedIn says an account can be used: the sync pane is neither signed out nor in the middle
// of signing in.
func (m *Main) voiceSignedIn() bool {
	if m.app.Sync == nil {
		return false
	}
	switch m.app.Sync.SyncSnapshot().Phase {
	case data.SyncSignedOut, data.SyncSigningIn, data.SyncBothHaveTasks:
		return false
	}
	return true
}

// voiceFail ends a try with the line that words why.
func (m *Main) voiceFail(p voiceProblem) {
	v := &m.voice
	v.phase, v.problem, v.problemAt = voiceIdle, p, m.app.Now()
}

// voiceToggle is a click on the microphone: it starts a recording, or ends the one that runs. A
// click while the text is on its way does nothing; Esc cancels that.
func (m *Main) voiceToggle(s surface) {
	switch m.voice.phase {
	case voiceListening:
		m.voiceStop()
	case voiceIdle:
		m.voiceBegin(s)
	}
}

// voiceBegin checks what is needed, asks for the microphone if the person has not been asked, and
// starts recording.
func (m *Main) voiceBegin(s surface) {
	v := &m.voice
	v.problem = voiceNoProblem
	if !m.voiceSignedIn() {
		m.voiceFail(voiceNeedSignIn)
		return
	}
	rec := m.app.Recorder
	switch rec.Authorization() {
	case speech.AuthDenied:
		m.voiceFail(voiceMicOff)
	case speech.AuthNotDetermined:
		v.gen++
		gen, post := v.gen, m.app.Act.Post
		rec.RequestAuthorization(func(a speech.Authorization) {
			post(func() {
				if gen != v.gen || v.phase != voiceIdle {
					return
				}
				switch a {
				case speech.AuthAuthorized:
					m.voiceRecord(s)
				case speech.AuthDenied:
					m.voiceFail(voiceMicOff)
				}
			})
		})
	default:
		m.voiceRecord(s)
	}
}

// voiceRecord starts the recorder. The composer is not given the focus: a press on the button took it
// from the field already, and the box must not change under the pointer before the second click.
func (m *Main) voiceRecord(s surface) {
	v := &m.voice
	err := m.app.Recorder.Start(v.noteLevel)
	switch {
	case errors.Is(err, speech.ErrNotAuthorized):
		m.voiceFail(voiceMicOff)
		return
	case err != nil:
		m.voiceFail(voiceCouldNot)
		return
	}
	now := m.app.Now()
	v.phase, v.owner, v.startedAt = voiceListening, s, now
	v.bars, v.beat = [theme.WaveBars]float32{}, now.UnixMilli()/theme.SpinnerStep.Milliseconds()
	v.peak.Store(0)
}

// voiceStop ends the recording and sends it to be transcribed, off the UI thread.
func (m *Main) voiceStop() {
	v := &m.voice
	if v.phase != voiceListening {
		return
	}
	wav, err := m.app.Recorder.Stop()
	if err != nil || m.app.Transcriber == nil {
		m.voiceFail(voiceCouldNot)
		return
	}
	v.gen++
	gen, owner, post, tr := v.gen, v.owner, m.app.Act.Post, m.app.Transcriber
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	v.phase, v.cancel = voiceTranscribing, cancel
	go func() {
		defer cancel()
		text, err := tr.Transcribe(ctx, wav)
		post(func() { m.transcribed(gen, owner, text, err) })
	}()
}

// transcribed runs on the UI thread when the server answered: the text goes into the box and is
// sent the way Enter sends it.
func (m *Main) transcribed(gen int, s surface, text string, err error) {
	v := &m.voice
	if gen != v.gen || v.phase != voiceTranscribing {
		return // cancelled meanwhile
	}
	v.phase, v.cancel = voiceIdle, nil
	switch {
	case err == nil:
	case errors.Is(err, syncclient.ErrVoiceNotSignedIn):
		m.voiceFail(voiceNeedSignIn)
		return
	case errors.Is(err, syncclient.ErrVoiceQuota):
		m.voiceFail(voiceQuotaHit)
		return
	case errors.Is(err, syncclient.ErrVoiceOff):
		m.voiceFail(voiceNotSetUp)
		return
	default:
		m.voiceFail(voiceCouldNot)
		return
	}
	cs := &m.cmp[s]
	cs.buf = strings.TrimSpace(cs.buf + " " + text)
	m.submit(s)
	m.refocus(s)
}

// voiceCancel drops the recording or the upload, and says whether there was one. Esc does it.
func (m *Main) voiceCancel() bool {
	v := &m.voice
	switch v.phase {
	case voiceListening:
		m.app.Recorder.Cancel()
	case voiceTranscribing:
		if v.cancel != nil {
			v.cancel()
		}
	default:
		return false
	}
	v.phase, v.cancel = voiceIdle, nil
	v.gen++
	return true
}

// voiceButton is the microphone at the right of the page's composer: a 24 pt pixel microphone, and
// while it records a filled accent square with the microphone in white. The text is on its way after
// a click, and the button waits.
func (m *Main) voiceButton(c *ui.Context, t *theme.Tokens, s surface) {
	if s != surfacePage || !m.voiceReady() {
		return
	}
	v := &m.voice
	listening := v.phase == voiceListening
	label := m.say.ComposerVoice()
	if listening {
		label = m.say.ComposerVoiceStop()
	}
	btn := ui.ButtonBase(c.Key("voice")).Size(voiceButtonSize, voiceButtonSize).Shrink(0).Center().
		AlignSelf(ui.End).Radius(8).Label(label).Tooltip(label)
	color := t.TextSecondary
	switch {
	case listening && btn.Hovered():
		btn.Background(t.Accent.Mix(t.Text, 0.12)).Cursor(ui.CursorPointer)
		color = t.AccentText
	case listening:
		btn.Background(t.Accent).Cursor(ui.CursorPointer)
		color = t.AccentText
	case v.phase == voiceTranscribing:
		color = t.TextTertiary
	case btn.Hovered():
		btn.Background(t.BgHover).Cursor(ui.CursorPointer)
		color = t.Text
	default:
		btn.Cursor(ui.CursorPointer)
	}
	btn.Transition(ui.ElementTransition{Colors: true, Duration: theme.MotionFast})
	btn.Children(func() { theme.VoiceMic(c, voiceMicSize, color) })
	if btn.Clicked() {
		m.voiceToggle(s)
	}
}

// voiceLine is the quiet line above the box: the waveform while listening, the busy indicator while
// transcribing, or why the last try failed. It also steps the waveform and ends a recording at the
// limit.
func (m *Main) voiceLine(c *ui.Context, t *theme.Tokens) {
	v := &m.voice
	now := m.app.Now()
	row := func(children func()) {
		ui.Row(c.Key("voice-line")).Height(answerLineHeight).Shrink(0).Margin(8, 0, 0).Padding(0, 12, 0, 16).Gap(10).
			AlignItems(ui.Center).Children(children)
	}
	switch v.phase {
	case voiceListening:
		if now.Sub(v.startedAt) >= speech.MaxDuration {
			m.voiceStop()
			return
		}
		if beat := now.UnixMilli() / theme.SpinnerStep.Milliseconds(); beat != v.beat {
			copy(v.bars[:], v.bars[1:])
			v.bars[len(v.bars)-1] = math.Float32frombits(v.peak.Swap(0))
			v.beat = beat
		}
		c.After(theme.SpinnerStep)
		row(func() {
			theme.PixelWave(c, v.bars[:], t.Accent)
			theme.Text(c, theme.Compact, t.TextSecondary, m.say.VoiceListening())
		})
	case voiceTranscribing:
		c.After(theme.SpinnerStep)
		row(func() {
			theme.PixelSpinner(c, theme.IconSmall, t.TextSecondary, theme.SpinnerFrame(now, c.Preferences().ReduceMotion))
			theme.Text(c, theme.Compact, t.TextSecondary, m.say.VoiceTranscribing())
		})
	default:
		m.voiceProblemLine(c, t, now, row)
	}
}

// voiceProblemLine is the line about a failure, for a few seconds. A missing account or microphone
// permission has a link to where it is fixed.
func (m *Main) voiceProblemLine(c *ui.Context, t *theme.Tokens, now time.Time, row func(func())) {
	v := &m.voice
	age := now.Sub(v.problemAt)
	if v.problem == voiceNoProblem || age >= voiceProblemFor {
		return
	}
	c.After(voiceProblemFor - age)
	switch v.problem {
	case voiceNeedSignIn:
		row(func() {
			if l := link(c, t, m.say.VoiceSignIn()); l.Clicked() {
				m.openSettings(PaneAccount)
			}
		})
	case voiceMicOff:
		row(func() {
			theme.Text(c, theme.Compact, t.TextSecondary, m.say.VoiceMicDenied())
			if l := link(c, t, m.say.VoiceOpenSystemSettings()); l.Clicked() {
				m.app.Act.OpenURL(voiceSettingsURL)
			}
		})
	default:
		text := m.say.VoiceFailed()
		switch v.problem {
		case voiceQuotaHit:
			text = m.say.VoiceQuota()
		case voiceNotSetUp:
			text = m.say.VoiceOff()
		}
		row(func() { theme.Text(c, theme.Compact, t.TextSecondary, text) })
	}
}
