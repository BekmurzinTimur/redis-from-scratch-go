package main

import (
	"slices"
	"sync"
	"time"
)

type Store struct {
	mu      sync.RWMutex
	strings map[string]string
	lists   map[string][]string
	expiry  map[string]time.Time
}

func NewStore() *Store {
	return &Store{strings: map[string]string{}, lists: map[string][]string{}, expiry: map[string]time.Time{}}
}

var store = NewStore()

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.strings[key]
	return v, ok
}

func (s *Store) Set(key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.strings[key] = value
}

func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.strings, key)
}

func (s *Store) AppendList(key string, items []string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lists[key] = append(s.lists[key], items...)
	length := len(s.lists[key])
	return length
}

func (s *Store) PrependList(key string, items []string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	slices.Reverse(items)
	s.lists[key] = append(items, s.lists[key]...)
	length := len(s.lists[key])
	return length
}

func (s *Store) GetList(key string, start int, stop int) ([]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list, ok := s.lists[key]
	length := len(list)
	if start < 0 {
		start = max(0, start+length)
	}
	if stop < 0 {
		stop = stop + length
		if stop < 0 {
			return []string{}, ok
		}
	}
	if start >= length {
		return []string{}, ok
	}
	if start > stop {
		return []string{}, ok
	}

	return list[start:min(stop+1, length)], ok
}

func (s *Store) SetExpiry(key string, expiryDate time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expiry[key] = expiryDate
}

func (s *Store) GetExpiry(key string) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.expiry[key]
	return v, ok
}

func (s *Store) DeleteExpiry(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.expiry, key)
}
