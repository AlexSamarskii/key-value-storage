package command

import (
	"keyvalue/internal/usecase/resp"
	"keyvalue/internal/usecase/storage"
	"strconv"
	"strings"
	"time"
)

func ExecuteCommand(cmd resp.Value, store *storage.Storage) resp.Value {
	command := strings.ToUpper(cmd.Array[0].Bulk)
	args := cmd.Array[1:]

	switch command {
	case "PING":
		return ping(args)
	case "GET":
		return get(args, store)
	case "SET":
		return set(args, store)
	case "DEL":
		return del(args, store)
	default:
		return resp.Value{Typ: "error", Str: "Unknown Command"}
	}
}

func ping(args []resp.Value) resp.Value {
	if len(args) == 0 {
		return resp.Value{Typ: "string", Str: "PONG"}
	}

	return resp.Value{Typ: "string", Str: args[0].Bulk}
}

func get(args []resp.Value, store *storage.Storage) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: "error", Str: "Usage: GET [key]"}
	}

	value, found := store.Get(args[0].Bulk)
	if !found {
		return resp.Value{Typ: "null"}
	}

	return resp.Value{Typ: "bulk", Bulk: value}
}

func set(args []resp.Value, store *storage.Storage) resp.Value {
	if len(args) < 2 || len(args) > 3 {
		return resp.Value{Typ: "error", Str: "Usage: SET [key] [value] [TTL]"}
	}

	key := args[0].Bulk
	value := args[1].Bulk
	var ttl time.Duration
	if len(args) == 3 {
		ttlSeconds, err := strconv.Atoi(args[2].Bulk)
		if err != nil {
			return resp.Value{Typ: "error", Str: "Invalid TTL value"}
		}

		ttl = time.Duration(ttlSeconds) * time.Second
	}

	store.Set(key, value, ttl)
	return resp.Value{Typ: "string", Str: "OK"}
}

func del(args []resp.Value, store *storage.Storage) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: "error", Str: "Usage: DEL [key]"}
	}

	err := store.Delete(args[0].Bulk)
	if err != nil {
		return resp.Value{Typ: "error", Str: err.Error()}
	}

	return resp.Value{Typ: "string", Str: "OK"}
}
