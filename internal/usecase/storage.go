package usecase

import "keyvalue/internal/usecase/storage"

type Storage interface {
	Get(key string) (storage.Value, error)
	Delete(key string) error
	Set(key string, value storage.Value) error
	Has(key string) bool
	Keys() []string
	Clear() error
	Close() error
}
