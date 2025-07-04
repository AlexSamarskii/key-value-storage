package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DiskStorage struct {
	path string
	mem  *MemoryStorage
	mx   sync.RWMutex
}

func NewDiskStorage(dir string) (*DiskStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	ds := &DiskStorage{
		path: dir,
		mem:  NewMemoryStorage(),
	}

	if err := ds.loadFromDisk(); err != nil {
		return nil, err
	}
	return ds, nil
}

func (d *DiskStorage) loadFromDisk() error {
	d.mx.Lock()
	defer d.mx.Unlock()

	files, err := os.ReadDir(d.path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filepath := filepath.Join(d.path, file.Name())
		data, err := os.ReadFile(filepath)
		if err != nil {
			continue
		}

		var value Value
		if err := json.Unmarshal(data, &value); err != nil {
			continue
		}
		if value.Expiration > 0 && value.Expiration < time.Now().UnixNano() {
			os.Remove(filepath)
			continue
		}
		d.mem.Set(file.Name(), value)
	}
	return nil
}

func (d *DiskStorage) Get(key string) (Value, error) {
	val, err := d.mem.Get(key)
	if err == nil {
		return val, nil
	}

	d.mx.RLock()
	defer d.mx.RUnlock()

	filepath := filepath.Join(d.path, key)
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return Value{}, ErrKeyNotFound
		}
		return Value{}, err
	}

	var value Value
	if err := json.Unmarshal(data, &value); err != nil {
		return Value{}, err
	}

	if value.Expiration > 0 && value.Expiration < time.Now().UnixNano() {
		d.mx.RLock()
		delete(d.mem.data, key)
		d.mx.RUnlock()
		return Value{}, ErrKeyExpired
	}

	return value, nil
}

func (d *DiskStorage) Set(key string, value Value) error {
	if err := d.mem.Set(key, value); err != nil {
		return err
	}

	d.mx.Lock()
	defer d.mx.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	filepath := filepath.Join(d.path, key)
	return os.WriteFile(filepath, data, 0644)
}

func (d *DiskStorage) Delete(key string) error {
	d.mem.Delete(key)

	d.mx.Lock()
	defer d.mx.Unlock()

	filepath := filepath.Join(d.path, key)
	err := os.Remove(filepath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (d *DiskStorage) Has(key string) bool {
	if d.mem.Has(key) {
		return true
	}

	d.mx.Lock()
	defer d.mx.Unlock()

	filepath := filepath.Join(d.path, key)
	_, err := os.Stat(filepath)
	if err != nil {
		return false
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		return false
	}

	var value Value
	if err := json.Unmarshal(data, &value); err != nil {
		return false
	}
	if value.Expiration > 0 && value.Expiration < time.Now().UnixNano() {
		d.mx.RLock()
		d.Delete(key)
		d.mx.RUnlock()
		return false
	}

	d.mem.Set(key, value)
	return true
}

func (d *DiskStorage) Keys() []string {
	if err := d.loadFromDisk(); err != nil {
		return d.mem.Keys()
	}
	return d.mem.Keys()
}

func (d *DiskStorage) Clear() error {
	d.mem.Clear()

	d.mx.Lock()
	defer d.mx.Unlock()

	files, err := os.ReadDir(d.path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(d.path, file.Name())
		if err := os.Remove(filePath); err != nil {
			return err
		}
	}

	return nil
}

func (d *DiskStorage) Close() error {
	return nil
}
