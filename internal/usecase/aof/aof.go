package aof

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

type Aof struct {
	file   *os.File
	reader *bufio.Reader
	mu     sync.Mutex
}

func NewAof(path string) (*Aof, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return nil, err
	}

	aof := &Aof{
		file:   f,
		reader: bufio.NewReader(f),
	}
	go func() {
		for {
			aof.mu.Lock()
			aof.file.Sync()
			aof.mu.Unlock()
			time.Sleep(time.Second)
		}
	}()

	return aof, nil
}

func (aof *Aof) Write(value string) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	val, _ := json.Marshal(value)
	_, err := aof.file.Write(val)
	if err != nil {
		return err
	}

	return nil
}

func (aof *Aof) Read(callback func(value string)) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	reader := resp.NewReader(aof.file)

	for {
		value, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		callback(value)
	}

	return nil
}
