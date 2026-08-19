package service

import (
	"context"
	"sync"
)

// settingRepoStub is a shared in-memory fake of SettingRepository used by
// multiple service test files. It lives in its own file (no build tag) so it is
// available to both unit-tagged tests and always-compiled tests.
//
// Methods use pointer receivers so the call counters stay observable through the
// interface; construct it as &settingRepoStub{...}.
type settingRepoStub struct {
	mu               sync.Mutex
	values           map[string]string
	err              error
	getValueCalls    int
	getMultipleCalls int
}

func (s *settingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *settingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getValueCalls++
	if s.err != nil {
		return "", s.err
	}
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *settingRepoStub) Set(context.Context, string, string) error { return nil }

func (s *settingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getMultipleCalls++
	if s.err != nil {
		return nil, s.err
	}
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *settingRepoStub) SetMultiple(context.Context, map[string]string) error { return nil }

func (s *settingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *settingRepoStub) Delete(context.Context, string) error { return nil }
