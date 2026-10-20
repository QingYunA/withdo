package main

import (
	"errors"

	"github.com/QingYunA/adhd-todo/internal/providers"
	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/egoist/mygo"
)

// syncSecrets is the syncclient.Secrets backed by mygo.Secrets: the login Keychain on macOS, under
// the same service as the model keys. The app injects it into data.Real and the command line, so
// internal/ never imports mygo.
type syncSecrets struct{ secrets mygo.SecretsModule }

// newSyncSecrets opens the keychain service the model keys already use.
func newSyncSecrets() syncSecrets {
	return syncSecrets{secrets: mygo.Secrets.Service(providers.KeychainService)}
}

// Get implements syncclient.Secrets.
func (s syncSecrets) Get(key string) (string, error) {
	raw, err := s.secrets.Get(key)
	if errors.Is(err, mygo.ErrSecretNotFound) {
		return "", syncclient.ErrSecretNotFound
	}
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// Set implements syncclient.Secrets.
func (s syncSecrets) Set(key, value string) error { return s.secrets.Set(key, []byte(value)) }

// Delete implements syncclient.Secrets; deleting a key that is not there is not an error.
func (s syncSecrets) Delete(key string) error {
	err := s.secrets.Delete(key)
	if errors.Is(err, mygo.ErrSecretNotFound) {
		return nil
	}
	return err
}
