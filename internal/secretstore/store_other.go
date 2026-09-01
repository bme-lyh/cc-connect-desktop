//go:build !windows

package secretstore

import "fmt"

type unavailableStore struct {
	service string
}

func Open(service string) Store {
	return &unavailableStore{service: service}
}

func (s *unavailableStore) Set(_, _ string) error {
	return fmt.Errorf("system credential store for %s is not available on this platform", s.service)
}

func (s *unavailableStore) Get(_ string) (string, error) {
	return "", fmt.Errorf("system credential store for %s is not available on this platform", s.service)
}

func (s *unavailableStore) Delete(_ string) error {
	return fmt.Errorf("system credential store for %s is not available on this platform", s.service)
}
