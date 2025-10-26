package util

import "sync"

type SyncSlice[T any] struct {
	mu    sync.RWMutex
	slice []T
}

func NewSyncSlice[T any]() *SyncSlice[T] {
	return &SyncSlice[T]{}
}

func (s *SyncSlice[T]) Add(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.slice = append(s.slice, item)
}

func (s *SyncSlice[T]) Items() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.slice
}
