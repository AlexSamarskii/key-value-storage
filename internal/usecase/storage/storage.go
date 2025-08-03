package storage

import (
	"errors"
	"sync"
	"time"
)

type Storage struct {
	data         map[string]string
	expiration   map[string]time.Time
	hCollections map[string]*NestedCollection
	mu           sync.RWMutex
	stopCleaner  chan struct{}
}

type NestedCollection struct {
	mu         sync.RWMutex
	fields     map[string]string
	expiration map[string]time.Time
}

func NewStorage() *Storage {
	store := &Storage{
		data:         make(map[string]string),
		expiration:   make(map[string]time.Time),
		hCollections: make(map[string]*NestedCollection),
		stopCleaner:  make(chan struct{}),
	}

	go store.startBackgroundCleaner()
	return store
}

// Stop очищает ресурсы хранилища
func (s *Storage) Stop() {
	close(s.stopCleaner)
}

func (s *Storage) startBackgroundCleaner() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanExpired()
		case <-s.stopCleaner:
			return
		}
	}
}

func (s *Storage) cleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Очистка основных ключей
	for key, expTime := range s.expiration {
		if now.After(expTime) {
			delete(s.data, key)
			delete(s.expiration, key)
		}
	}

	// Очистка вложенных коллекций
	for _, coll := range s.hCollections {
		for field, expTime := range coll.expiration {
			if now.After(expTime) {
				delete(coll.fields, field)
				delete(coll.expiration, field)
			}
		}
	}
}

// Set сохраняет значение с опциональным TTL
func (s *Storage) Set(key string, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
	if ttl > 0 {
		s.expiration[key] = time.Now().Add(ttl)
	} else {
		delete(s.expiration, key)
	}
}

// Get возвращает значение по ключу
func (s *Storage) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if expTime, exists := s.expiration[key]; exists && time.Now().After(expTime) {
		return "", false
	}

	value, found := s.data[key]
	return value, found
}

// Delete удаляет ключ из хранилища
func (s *Storage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, found := s.data[key]; !found {
		return errors.New("key not found")
	}

	delete(s.data, key)
	delete(s.expiration, key)
	return nil
}

// HCollection создает или возвращает вложенную коллекцию
func (s *Storage) HCollection(name string) *NestedCollection {
	s.mu.Lock()
	defer s.mu.Unlock()

	if coll, exists := s.hCollections[name]; exists {
		return coll
	}

	coll := &NestedCollection{
		fields:     make(map[string]string),
		expiration: make(map[string]time.Time),
	}
	s.hCollections[name] = coll
	return coll
}

// HSet устанавливает значение в вложенной коллекции
func (s *Storage) HSet(collection, field string, value string, ttl time.Duration) {
	coll := s.HCollection(collection)
	coll.mu.Lock()
	defer coll.mu.Unlock()

	coll.fields[field] = value
	if ttl > 0 {
		coll.expiration[field] = time.Now().Add(ttl)
	} else {
		delete(coll.expiration, field)
	}
}

// HGet получает значение из вложенной коллекции
func (s *Storage) HGet(collection, field string) (string, bool) {
	s.mu.RLock()
	coll, exists := s.hCollections[collection]
	s.mu.RUnlock()

	if !exists {
		return "", false
	}

	coll.mu.RLock()
	defer coll.mu.RUnlock()

	if expTime, exists := coll.expiration[field]; exists && time.Now().After(expTime) {
		return "", false
	}

	value, found := coll.fields[field]
	return value, found
}

// HDelete удаляет поле из вложенной коллекции
func (s *Storage) HDelete(collection, field string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	coll, exists := s.hCollections[collection]
	if !exists {
		return errors.New("collection not found")
	}

	if _, found := coll.fields[field]; !found {
		return errors.New("field not found")
	}

	delete(coll.fields, field)
	delete(coll.expiration, field)
	return nil
}

// HDeleteAll удаляет всю вложенную коллекцию
func (s *Storage) HDeleteAll(collection string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.hCollections[collection]; !exists {
		return errors.New("collection not found")
	}

	delete(s.hCollections, collection)
	return nil
}

// Exists проверяет существование ключа
func (s *Storage) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if expTime, exists := s.expiration[key]; exists && time.Now().After(expTime) {
		return false
	}

	_, found := s.data[key]
	return found
}

// TTL возвращает оставшееся время жизни ключа
func (s *Storage) TTL(key string) (time.Duration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	expTime, exists := s.expiration[key]
	if !exists {
		return 0, errors.New("key has no TTL")
	}

	remaining := time.Until(expTime)
	if remaining <= 0 {
		return 0, errors.New("key expired")
	}

	return remaining, nil
}

func (s *Storage) SetTTL(key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, found := s.data[key]
	if !found {
		return errors.New("key not found")
	}

	if ttl > 0 {
		s.expiration[key] = time.Now().Add(ttl)
	} else {
		// Если ttl == 0 или < 0, удаляем TTL (бессрочный ключ)
		delete(s.expiration, key)
	}

	return nil
}
