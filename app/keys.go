package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/providers"
	"github.com/egoist/mygo"
)

// Prefixes go-keyring put in front of the values it stored (this app's earlier builds used it).
const (
	legacyBase64Prefix = "go-keyring-base64:"
	legacyHexPrefix    = "go-keyring-encoded:"
)

// secretKeyring is the providers.Keyring backed by mygo.Secrets: the login Keychain on macOS.
// Same service and account as the go-keyring one it replaces, so stored keys are found.
type secretKeyring struct{ secrets mygo.SecretsModule }

func newSecretKeyring() secretKeyring {
	return secretKeyring{secrets: mygo.Secrets.Service(providers.KeychainService)}
}

func secretAccount(id string) string { return "model/" + id }

// Get implements providers.Keyring. A value go-keyring stored is decoded and written back as the
// plain key, best effort, so the next read is plain.
func (k secretKeyring) Get(id string) (string, error) {
	raw, err := k.secrets.Get(secretAccount(id))
	if errors.Is(err, mygo.ErrSecretNotFound) {
		return "", providers.ErrNoKey
	}
	if err != nil {
		return "", err
	}
	key, legacy, err := decodeLegacySecret(string(raw))
	if err != nil {
		return "", fmt.Errorf("stored key for %s: %w", id, err)
	}
	if legacy {
		if err := k.secrets.Set(secretAccount(id), []byte(key)); err != nil {
			log.Printf("keys: rewriting the stored key for %s: %v", id, err)
		}
	}
	return key, nil
}

// Set implements providers.Keyring.
func (k secretKeyring) Set(id, key string) error {
	return k.secrets.Set(secretAccount(id), []byte(key))
}

// Delete implements providers.Keyring; deleting a key that is not there is not an error.
func (k secretKeyring) Delete(id string) error { return k.secrets.Delete(secretAccount(id)) }

// decodeLegacySecret returns the plain value of a stored secret. legacy reports that it carried a
// go-keyring prefix, so the caller can rewrite it. A value without a prefix is returned as is.
func decodeLegacySecret(raw string) (key string, legacy bool, err error) {
	raw = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(raw, legacyBase64Prefix):
		b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, legacyBase64Prefix))
		return string(b), true, err
	case strings.HasPrefix(raw, legacyHexPrefix):
		b, err := hex.DecodeString(strings.TrimPrefix(raw, legacyHexPrefix))
		return string(b), true, err
	}
	return raw, false, nil
}
