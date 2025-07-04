package raft

import (
	"encoding/json"
	command "keyvalue/internal/pkg"
	"keyvalue/internal/usecase/storage"

	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

type FSM struct {
	store  storage.MemoryStorage
	logger *zap.Logger
}

type fsmSnapshot struct {
	data map[string]storage.Value
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := json.NewEncoder(sink).Encode(s.data)
	if err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

// noop
func (s *fsmSnapshot) Release() {}

func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd command.Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		f.logger.Error("Failed to unmarshal command", zap.Error(err))
		return err
	}

	switch cmd.Op {
	case "set":
		err := f.store.Set(cmd.Key, cmd.Value)
		if err != nil {
			f.logger.Error("Failed to set value", zap.String("key", cmd.Key), zap.Error(err))
			return err
		}
		f.logger.Debug("Set value", zap.String("key", cmd.Key))
		return nil

	case "delete":
		err := f.store.Delete(cmd.Key)
		if err != nil {
			f.logger.Error("Failed to delete key", zap.String("key", cmd.Key), zap.Error(err))
			return err
		}
		f.logger.Debug("Deleted key", zap.String("key", cmd.Key))
		return nil

	case "delete-all":
		err := f.store.Clear()
		if err != nil {
			f.logger.Error("Failed to clear store", zap.Error(err))
			return err
		}
		f.logger.Debug("Cleared store")
		return nil

	default:
		err := json.Unmarshal(log.Data, &cmd)
		f.logger.Error("Unknown command", zap.String("op", cmd.Op), zap.Error(err))
		return err
	}
}

func (f *FSM) getSnapshot() (raft.FSMSnapshot, error) {
	f.logger.Debug("Creating snapshot")
	keys := f.store.Keys()
	data := make(map[string]storage.Value, len(keys))

	for _, key := range keys {
		value, err := f.store.Get(key)
		if err == nil {
			data[key] = value
		}
	}

	return &fsmSnapshot{data: data}, nil
}
