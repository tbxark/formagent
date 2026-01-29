package agent

import (
	"context"
	"sync"
)

type Cache[S any] interface {
	Set(ctx context.Context, key string, val S) error
	Get(ctx context.Context, key string) (S, bool, error)
	Del(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type MemoryCache[S any] struct {
	mu sync.RWMutex
	m  map[string]S
}

func NewMemoryCore[S any]() *MemoryCache[S] {
	return &MemoryCache[S]{m: map[string]S{}}
}

func (m *MemoryCache[S]) Set(ctx context.Context, key string, val S) error {
	m.mu.Lock()
	m.m[key] = val
	m.mu.Unlock()
	return nil
}

func (m *MemoryCache[S]) Get(ctx context.Context, key string) (S, bool, error) {
	m.mu.RLock()
	val, ok := m.m[key]
	m.mu.RUnlock()
	return val, ok, nil
}

func (m *MemoryCache[S]) Del(ctx context.Context, key string) error {
	m.mu.Lock()
	delete(m.m, key)
	m.mu.Unlock()
	return nil
}

func (m *MemoryCache[S]) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	_, ok := m.m[key]
	m.mu.RUnlock()
	return ok, nil
}
