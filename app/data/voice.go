package data

import (
	"context"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// Transcriber turns a recording (16 kHz mono PCM16 WAV) into text through the sync server. Its
// errors are the syncclient.ErrVoice* sentinels.
type Transcriber interface {
	Transcribe(ctx context.Context, wav []byte) (string, error)
}

// Transcribe uploads a recording with the signed-in account's session. It is off the UI thread by
// the caller. Without an account it fails with syncclient.ErrVoiceNotSignedIn.
func (r *Real) Transcribe(ctx context.Context, wav []byte) (string, error) {
	r.mu.Lock()
	r.loadSyncLocked()
	client, cfg := r.sync.client, r.sync.cfg
	r.mu.Unlock()
	if client == nil {
		if cfg.Server == "" {
			return "", syncclient.ErrVoiceNotSignedIn
		}
		var err error
		if client, err = r.syncClientFor(cfg); err != nil {
			return "", syncclient.ErrVoiceNotSignedIn
		}
	}
	return client.Transcribe(ctx, wav)
}
