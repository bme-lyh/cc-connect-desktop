package secretstore

import (
	"fmt"
	"strings"
	"sync"
)

const refPrefix = "secret://cc-connect/"

type Store interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
}

func Reference(key string) string {
	return refPrefix + strings.TrimPrefix(key, "/")
}

func ParseReference(value string) (string, bool) {
	if !strings.HasPrefix(value, refPrefix) {
		return "", false
	}
	key := strings.TrimPrefix(value, refPrefix)
	return key, key != ""
}

type MemoryStore struct {
	mu     sync.RWMutex
	values map[string]string
}

func NewMemory() *MemoryStore {
	return &MemoryStore{values: make(map[string]string)}
}

func (m *MemoryStore) Set(key, value string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("secret key is required")
	}
	m.mu.Lock()
	m.values[key] = value
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) Get(key string) (string, error) {
	m.mu.RLock()
	value, ok := m.values[key]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("secret %q not found", key)
	}
	return value, nil
}

func (m *MemoryStore) Delete(key string) error {
	m.mu.Lock()
	delete(m.values, key)
	m.mu.Unlock()
	return nil
}
