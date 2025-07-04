package pkg

import (
	"keyvalue/internal/usecase/storage"
)

type Command struct {
	Op    string        `json:"op"`    // "set", "delete", "deleteAll"
	Key   string        `json:"key"`   // Key to operate on
	Value storage.Value `json:"value"` // Value for set operation
}
