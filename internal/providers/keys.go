package providers

import (
	"errors"
	"sync"

	"github.com/zalando/go-keyring"
)

// ErrNoKey means no API key is stored for the provider.
var ErrNoKey = errors.New("no API key stored")

// KeychainService is the service name the app's keys are filed under.
const KeychainService = "adhd-todo"

// Keyring stores one API key per provider id. Keys never go anywhere else: not the database, not
// the provider file, not the logs.
type Keyring interface {
	Get(id string) (string, error) // ErrNoKey when there is none
	Set(id, key string) error
	Delete(id string) error
}

// Keychain is the Keyring backed by the system keychain (macOS Keychain, Secret Service, Windows
// Credential Manager) under Service.
type Keychain struct{ Service string }

// NewKeychain files keys under KeychainService.
func NewKeychain() Keychain { return Keychain{Service: KeychainService} }

func (k Keychain) user(id string) string { return "model/" + id }

// Get implements Keyring.
func (k Keychain) Get(id string) (string, error) {
	v, err := keyring.Get(k.Service, k.user(id))
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNoKey
	}
	return v, err
}

// Set implements Keyring.
func (k Keychain) Set(id, key string) error { return keyring.Set(k.Service, k.user(id), key) }

// Delete implements Keyring; deleting a key that is not there is not an error.
func (k Keychain) Delete(id string) error {
	if err := keyring.Delete(k.Service, k.user(id)); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
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
