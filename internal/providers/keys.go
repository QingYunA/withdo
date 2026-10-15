package providers

import (
	"errors"
	"sync"
)

// ErrNoKey means no API key is stored for the provider.
var ErrNoKey = errors.New("no API key stored")

// KeychainService is the service name the app's keys are filed under in the system keychain
// (see app/keys.go).
const KeychainService = "adhd-todo"

// Keyring stores one API key per provider id. Keys never go anywhere else: not the database, not
// the provider file, not the logs.
type Keyring interface {
	Get(id string) (string, error) // ErrNoKey when there is none
	Set(id, key string) error
	Delete(id string) error
}

// MemoryKeyring is a Keyring in memory, for tests and the demo app.
type MemoryKeyring struct {
	mu   sync.Mutex
	keys map[string]string
}

// NewMemoryKeyring returns an empty one.
func NewMemoryKeyring() *MemoryKeyring { return &MemoryKeyring{keys: map[string]string{}} }

// Get implements Keyring.
func (m *MemoryKeyring) Get(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.keys[id]; ok {
		return v, nil
	}
	return "", ErrNoKey
}

// Set implements Keyring.
func (m *MemoryKeyring) Set(id, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[id] = key
	return nil
}

// Delete implements Keyring.
func (m *MemoryKeyring) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.keys, id)
	return nil
}
