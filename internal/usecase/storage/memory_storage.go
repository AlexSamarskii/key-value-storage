package storage

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrKeyNotFound = errors.New("Key not found")
	ErrKeyExpired  = errors.New("Key expired")
)

type Value struct {
	Data       []byte
	Expiration int64
}

type MemoryStorage struct {
	data map[string]Value
	mx   sync.RWMutex
}

func (m *MemoryStorage) Get(key string) (Value, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	val, ok := m.data[key]
	if !ok {
		return Value{}, ErrKeyNotFound
	}

	if val.Expiration > 0 && val.Expiration < time.Now().UnixNano() {
		m.mx.RUnlock()
		m.Delete(key)
		m.mx.RLock()
		return Value{}, ErrKeyExpired
	}
	return val, nil
}

func (m *MemoryStorage) Delete(key string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MemoryStorage) Set(key string, value Value) error {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.data[key] = value
	return nil
}

func (m *MemoryStorage) Has(key string) bool {
	m.mx.RLock()
	defer m.mx.RUnlock()

	val, ok := m.data[key]
	if !ok {
		return false
	}

	if val.Expiration > 0 && val.Expiration < time.Now().UnixNano() {
		return false
	}
	return true
}

func (m *MemoryStorage) Keys() []string {
	m.mx.RLock()
	defer m.mx.RUnlock()

	keys := make([]string, 0, len(m.data))
	now := time.Now().UnixNano()

	for k, v := range m.data {
		if v.Expiration > 0 && v.Expiration < now {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}

func (m *MemoryStorage) Clear() error {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.data = make(map[string]Value)
	return nil
}

func (m *MemoryStorage) Close() error {
	m.Clear()
	return nil
}
