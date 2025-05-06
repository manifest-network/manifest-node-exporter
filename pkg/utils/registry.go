package utils

import "sync"

type Registry[T any] struct {
	mu    sync.RWMutex
	store map[string]T
}

func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{
		store: make(map[string]T),
	}
}

func (r *Registry[T]) Register(key string, val T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[key] = val
}

func (r *Registry[T]) Get(key string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.store[key]
	return val, ok
}

func (r *Registry[T]) GetAll() map[string]T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	storeCopy := make(map[string]T, len(r.store))
	for k, v := range r.store {
		storeCopy[k] = v
	}
	return storeCopy
}
