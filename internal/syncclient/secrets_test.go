package syncclient_test

import (
	"sync"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// fakeSecrets is an in-memory syncclient.Secrets.
type fakeSecrets struct {
	mu     sync.Mutex
	values map[string]string
}

func newFakeSecrets() *fakeSecrets { return &fakeSecrets{values: map[string]string{}} }

func (s *fakeSecrets) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok {
		return "", syncclient.ErrSecretNotFound
	}
	return value, nil
}

func (s *fakeSecrets) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

func (s *fakeSecrets) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	return nil
}
