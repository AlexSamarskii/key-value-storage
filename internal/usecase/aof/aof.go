package aof

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"keyvalue/internal/usecase/resp"
	"os"
	"sync"
	"time"
)

var (
	ErrAofClosed = errors.New("AOF file is closed")
	SyncInterval = 1 * time.Second
)

type Aof struct {
	file     *os.File
	filePath string
	reader   *bufio.Reader
	writer   *bufio.Writer
	mu       sync.RWMutex
	closed   bool
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewAof(path string) (*Aof, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	aof := &Aof{
		file:     f,
		filePath: path,
		reader:   bufio.NewReader(f),
		writer:   bufio.NewWriter(f),
		stopChan: make(chan struct{}),
	}

	// Запускаем периодическую синхронизацию
	aof.wg.Add(1)
	go aof.syncLoop()

	return aof, nil
}

func (a *Aof) syncLoop() {
	defer a.wg.Done()

	for {
		select {
		case <-time.After(SyncInterval):
			a.Flush()
		case <-a.stopChan:
			a.Flush() // Последняя синхронизация перед выходом
			return
		}
	}
}

func (a *Aof) Write(value resp.Value) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return ErrAofClosed
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if _, err := a.writer.Write(append(data, '\n')); err != nil {
		return err
	}

	return nil
}

func (a *Aof) Flush() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return nil
	}

	if err := a.writer.Flush(); err != nil {
		return err
	}

	return a.file.Sync()
}

func (a *Aof) Read(callback func(value resp.Value)) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.closed {
		return ErrAofClosed
	}

	if _, err := a.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	decoder := json.NewDecoder(a.file)
	for {
		var value resp.Value
		if err := decoder.Decode(&value); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		callback(value)
	}

	return nil
}

func (a *Aof) Rewrite() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return ErrAofClosed
	}

	// Создаем временный файл для перезаписи
	tmpPath := a.filePath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	if _, err := a.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		return err
	}

	// Заменяем старый файл новым
	if err := os.Rename(tmpPath, a.filePath); err != nil {
		return err
	}

	if err := a.reopenFile(); err != nil {
		return err
	}

	return nil
}

func (a *Aof) reopenFile() error {
	if a.file != nil {
		a.file.Close()
	}

	f, err := os.OpenFile(a.filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	a.file = f
	a.reader = bufio.NewReader(f)
	a.writer = bufio.NewWriter(f)
	return nil
}

func (a *Aof) Close() error {
	a.mu.Lock()

	if a.closed {
		a.mu.Unlock()
		return nil
	}

	a.closed = true
	a.mu.Unlock()

	close(a.stopChan)
	a.wg.Wait()

	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.writer.Flush(); err != nil {
		return err
	}
	if err := a.file.Sync(); err != nil {
		return err
	}

	return a.file.Close()
}

func (a *Aof) Size() (int64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.closed {
		return 0, ErrAofClosed
	}

	stat, err := a.file.Stat()
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}
