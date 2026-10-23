package views

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/internal/speech"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/egoist/mygo/ui"
)

// The voice button of the composer: a click records, a second click (or Enter) sends the recording
// to be transcribed, and the text is put in the box for the person to edit and send. The recorder and the transcriber are fakes.

type fakeRecorder struct {
	mu        sync.Mutex
	absent    bool // Available says no
	auth      speech.Authorization
	startErr  error
	stopWAV   []byte
	stopErr   error
	started   int
	stopped   int
	cancelled int
	onLevel   func(float32)
	asked     func(speech.Authorization) // the permission prompt, until answered
}

func (f *fakeRecorder) Available() (bool, string) { return !f.absent, "" }
func (f *fakeRecorder) Authorization() speech.Authorization {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.auth
}
func (f *fakeRecorder) RequestAuthorization(done func(speech.Authorization)) {
	f.mu.Lock()
	f.asked = done
	f.mu.Unlock()
}
func (f *fakeRecorder) Start(onLevel func(float32)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.startErr != nil {
		return f.startErr
	}
	f.started++
	f.onLevel = onLevel
	return nil
}
func (f *fakeRecorder) Stop() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped++
	return f.stopWAV, f.stopErr
}
func (f *fakeRecorder) Cancel() {
	f.mu.Lock()
	f.cancelled++
	f.mu.Unlock()
}

// answerPrompt is the person answering the permission prompt.
func (f *fakeRecorder) answerPrompt(a speech.Authorization) {
	f.mu.Lock()
	done := f.asked
	f.auth = a
	f.mu.Unlock()
	done(a)
}

type fakeTranscriber struct {
	mu   sync.Mutex
	text string
	err  error
	gate chan struct{} // when non-nil, an upload waits for it to close or its context to end
	got  [][]byte
}

func (f *fakeTranscriber) Transcribe(ctx context.Context, wav []byte) (string, error) {
	f.mu.Lock()
	f.got = append(f.got, wav)
	gate, text, err := f.gate, f.text, f.err
	f.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return text, err
}

func (f *fakeTranscriber) uploads() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.got)
}

type voiceRig struct {
	*assistRig
	rc *fakeRecorder
	tr *fakeTranscriber
	sy *fakeSync
}

// newVoiceRig is a window signed in to sync, with an authorized microphone and a transcriber that
// hears "buy milk".
func newVoiceRig(t *testing.T) *voiceRig {
	t.Helper()
	r := newAssistRig(t, &data.FakeAssistant{Ready: true})
	v := &voiceRig{
		assistRig: r,
		rc:        &fakeRecorder{auth: speech.AuthAuthorized, stopWAV: []byte("RIFF-wav")},
		tr:        &fakeTranscriber{text: "buy milk"},
		sy:        &fakeSync{snap: syncedSync()},
	}
	r.app.Recorder, r.app.Transcriber, r.app.Sync = v.rc, v.tr, v.sy
	r.tt.Frame()
	return v
}

func (v *voiceRig) click(t *testing.T, label string) {
	t.Helper()
	if err := v.tt.Click(label); err != nil {
		t.Fatalf("click %q: %v in %q", label, err, v.tt.Texts())
	}
}

func TestTheMicrophoneIsHiddenWhereNothingCanRecord(t *testing.T) {
	v := newVoiceRig(t)
	if _, ok := v.tt.Find("Voice input"); !ok {
		t.Fatalf("no microphone button in %q", v.tt.Texts())
	}
	v.rc.absent = true
	v.tt.Frame()
	if _, ok := v.tt.Find("Voice input"); ok {
		t.Error("the microphone shows although the recorder is not available")
	}
	v.rc.absent = false
	v.app.Recorder = nil
	v.tt.Frame()
	if _, ok := v.tt.Find("Voice input"); ok {
		t.Error("the microphone shows without a recorder")
	}
}

func TestRecordStopTranscribeAndFill(t *testing.T) {
	v := newVoiceRig(t)
	v.click(t, "Voice input")
	if v.rc.started != 1 || v.m.voice.phase != voiceListening {
		t.Fatalf("started %d, phase %v", v.rc.started, v.m.voice.phase)
	}
	v.tt.Frame()
	has(t, v.tt, "Listening… click to stop, Esc to cancel")

	// The waveform steps every 100 ms with the loudest level since the last step.
	v.rc.onLevel(0.3)
	v.rc.onLevel(0.9)
	v.rc.onLevel(0.1)
	v.now = v.now.Add(100 * time.Millisecond)
	v.tt.Frame()
	v.rc.onLevel(0.5)
	v.now = v.now.Add(100 * time.Millisecond)
	v.tt.Frame()
	bars := v.m.voice.bars
	if bars[len(bars)-2] != 0.9 || bars[len(bars)-1] != 0.5 {
		t.Errorf("bars = %v, want the last two 0.9 and 0.5", bars)
	}

	v.tr.gate = make(chan struct{})
	v.click(t, "Stop listening")
	v.tt.Frame()
	if v.rc.stopped != 1 {
		t.Fatalf("stopped %d times", v.rc.stopped)
	}
	has(t, v.tt, "Transcribing…")
	if _, ok := v.tt.Find("Listening… click to stop, Esc to cancel"); ok {
		t.Error("the listening line stays while transcribing")
	}
	close(v.tr.gate)
	v.rec.drain(t)
	v.tt.Frame()
	v.tt.Frame()
	v.tt.Frame()
	if got := v.fa.Asked(); len(got) != 0 {
		t.Fatalf("the transcription was sent: %v", got)
	}
	if string(v.tr.got[0]) != "RIFF-wav" {
		t.Errorf("uploaded %q", v.tr.got[0])
	}
	if v.pageBuf() != "buy milk" || v.m.voice.phase != voiceIdle {
		t.Errorf("buf %q, phase %v", v.pageBuf(), v.m.voice.phase)
	}
	if !v.tt.Focused("Ask the assistant") {
		t.Error("the field is not focused")
	}
	v.tt.Type("!") // the caret is at the end of the text
	if got := v.pageBuf(); got != "buy milk!" {
		t.Errorf("buf %q after typing, want the caret at the end", got)
	}
}

func TestVoiceTextIsAddedAfterWhatWasTyped(t *testing.T) {
	v := newVoiceRig(t)
	v.tt.Key(ui.Cmd, ui.KeyK)
	v.tt.Type("remind me to")
	v.click(t, "Voice input")
	v.tt.Key(0, ui.KeyEnter) // ends the recording; nothing is sent yet
	if got := v.fa.Asked(); len(got) != 0 {
		t.Fatalf("Enter sent %v while recording", got)
	}
	if v.rc.stopped != 1 {
		t.Fatalf("Enter did not end the recording")
	}
	v.rec.drain(t)
	v.tt.Frame()
	if got := v.fa.Asked(); len(got) != 0 {
		t.Fatalf("asked %v", got)
	}
	if got := v.pageBuf(); got != "remind me to buy milk" {
		t.Fatalf("buf %q", got)
	}
}

func TestEscCancelsTheRecordingAndSendsNothing(t *testing.T) {
	v := newVoiceRig(t)
	v.click(t, "Voice input")
	v.tt.Key(0, ui.KeyEscape)
	if v.rc.cancelled != 1 || v.m.voice.phase != voiceIdle || v.tr.uploads() != 0 {
		t.Fatalf("cancelled %d, phase %v, uploads %d", v.rc.cancelled, v.m.voice.phase, v.tr.uploads())
	}
	v.tt.Frame()
	if _, ok := v.tt.Find("Listening… click to stop, Esc to cancel"); ok {
		t.Error("the listening line stays after Esc")
	}
}

func TestEscWhileTranscribingDropsTheAnswer(t *testing.T) {
	v := newVoiceRig(t)
	v.tr.gate = make(chan struct{})
	v.click(t, "Voice input")
	v.click(t, "Stop listening")
	v.tt.Key(0, ui.KeyEscape)
	if v.m.voice.phase != voiceIdle {
		t.Fatalf("phase %v", v.m.voice.phase)
	}
	close(v.tr.gate) // the answer comes late
	v.rec.drain(t)
	if got := v.fa.Asked(); len(got) != 0 {
		t.Errorf("a cancelled recording was sent: %v", got)
	}
	v.tt.Frame()
	if _, ok := v.tt.Find("Transcribing…"); ok {
		t.Error("Transcribing… stays after Esc")
	}
}

func TestLeavingThePageCancelsTheRecording(t *testing.T) {
	v := newVoiceRig(t)
	v.click(t, "Voice input")
	v.m.goTo(nav{screen: screenAgents})
	if v.rc.cancelled != 1 || v.m.voice.phase != voiceIdle {
		t.Fatalf("cancelled %d, phase %v", v.rc.cancelled, v.m.voice.phase)
	}
}

func TestARecordingEndsAtSixtySeconds(t *testing.T) {
	v := newVoiceRig(t)
	v.tr.gate = make(chan struct{})
	v.click(t, "Voice input")
	v.now = v.now.Add(speech.MaxDuration)
	v.tt.Frame()
	if v.rc.stopped != 1 || v.m.voice.phase != voiceTranscribing {
		t.Fatalf("stopped %d, phase %v", v.rc.stopped, v.m.voice.phase)
	}
	close(v.tr.gate)
	v.rec.drain(t)
}

func TestSignedOutSaysSoAndLinksToTheSyncPane(t *testing.T) {
	v := newVoiceRig(t)
	v.sy.snap.Phase = data.SyncSignedOut
	v.click(t, "Voice input")
	v.tt.Frame()
	has(t, v.tt, "Sign in to use voice")
	if v.rc.started != 0 {
		t.Error("recorded while signed out")
	}
	v.click(t, "Sign in to use voice")
	if len(v.rec.panes) != 1 || v.rec.panes[0] != PaneAccount {
		t.Errorf("settings opened %v", v.rec.panes)
	}
	v.now = v.now.Add(voiceProblemFor)
	v.tt.Frame()
	if _, ok := v.tt.Find("Sign in to use voice"); ok {
		t.Error("the line stays")
	}
}

func TestADeniedMicrophoneLinksToSystemSettings(t *testing.T) {
	v := newVoiceRig(t)
	v.rc.auth = speech.AuthDenied
	v.click(t, "Voice input")
	v.tt.Frame()
	has(t, v.tt, "Microphone access is off", "Open System Settings")
	v.click(t, "Open System Settings")
	if len(v.rec.urls) != 1 || v.rec.urls[0] != voiceSettingsURL {
		t.Errorf("opened %v", v.rec.urls)
	}
	if v.rc.started != 0 {
		t.Error("recorded without permission")
	}
}

func TestThePermissionPromptStartsTheRecordingWhenAllowed(t *testing.T) {
	v := newVoiceRig(t)
	v.rc.auth = speech.AuthNotDetermined
	v.click(t, "Voice input")
	if v.rc.asked == nil || v.rc.started != 0 {
		t.Fatalf("asked %v, started %d", v.rc.asked != nil, v.rc.started)
	}
	v.rc.answerPrompt(speech.AuthAuthorized)
	v.rec.drain(t)
	if v.rc.started != 1 || v.m.voice.phase != voiceListening {
		t.Fatalf("started %d, phase %v", v.rc.started, v.m.voice.phase)
	}
}

func TestRefusingThePermissionPromptShowsTheMicrophoneLine(t *testing.T) {
	v := newVoiceRig(t)
	v.rc.auth = speech.AuthNotDetermined
	v.click(t, "Voice input")
	v.rc.answerPrompt(speech.AuthDenied)
	v.rec.drain(t)
	v.tt.Frame()
	has(t, v.tt, "Microphone access is off")
}

func TestFailuresAreOneQuietLine(t *testing.T) {
	cases := []struct {
		name string
		err  error
		line string
	}{
		{"quota", syncclient.ErrVoiceQuota, "Voice limit reached for today"},
		{"not signed in", syncclient.ErrVoiceNotSignedIn, "Sign in to use voice"},
		{"server without voice", syncclient.ErrVoiceOff, "Voice isn't set up on this server"},
		{"network", syncclient.ErrVoiceNetwork, "Couldn't transcribe. Try again"},
		{"nothing heard", syncclient.ErrVoiceNoSpeech, "Couldn't transcribe. Try again"},
		{"other", errors.New("boom"), "Couldn't transcribe. Try again"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := newVoiceRig(t)
			v.tr.err = tc.err
			v.click(t, "Voice input")
			v.click(t, "Stop listening")
			v.rec.drain(t)
			v.tt.Frame()
			has(t, v.tt, tc.line)
			if got := v.fa.Asked(); len(got) != 0 {
				t.Errorf("sent %v after a failure", got)
			}
		})
	}
}

func TestARecorderThatCannotStartOrStopIsAGenericFailure(t *testing.T) {
	v := newVoiceRig(t)
	v.rc.startErr = errors.New("no input device")
	v.click(t, "Voice input")
	v.tt.Frame()
	has(t, v.tt, "Couldn't transcribe. Try again")

	v = newVoiceRig(t)
	v.rc.stopErr, v.rc.stopWAV = speech.ErrNoAudio, nil
	v.click(t, "Voice input")
	v.click(t, "Stop listening")
	v.tt.Frame()
	has(t, v.tt, "Couldn't transcribe. Try again")
	if v.tr.uploads() != 0 {
		t.Error("an empty recording was uploaded")
	}
}
