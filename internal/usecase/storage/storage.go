package storage

import (
	"sync"
	"time"
)

type Storage struct {
	data       map[string]string
	hset       map[string]map[string]string
	expiration map[string]time.Time
	mu         sync.RWMutex
}

func NewStorage() *Storage {
	store := &Storage{
		data:       make(map[string]string),
		hset:       make(map[string]map[string]string),
		expiration: make(map[string]time.Time),
	}

	go store.startExpirationCheck()

	return store
}

func (s *Storage) startExpirationCheck() {
	for {
		time.Sleep(time.Second)

		s.mu.Lock()
		now := time.Now()

		for key, expTime := range s.expiration {
			if now.After(expTime) {
				delete(s.data, key)
				delete(s.expiration, key)
			}
		}

		s.mu.Unlock()
	}
}

func (s *Storage) Set(key, value string, ttl time.Duration) {}

func (s *Storage) Get(key string) (string, bool) {
	return "", false
}

func (s *Storage) Delete(key string) error {
	return nil
}
