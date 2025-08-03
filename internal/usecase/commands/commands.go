package command

import (
	"keyvalue/internal/usecase/resp"
	"keyvalue/internal/usecase/storage"
	"strconv"
	"strings"
	"time"
)

type CommandExecutor struct {
	store     *storage.Storage
	commands  map[string]CommandHandler
	startTime time.Time
}

type CommandHandler func(args []resp.Value) resp.Value

func NewCommandExecutor(store *storage.Storage) *CommandExecutor {
	executor := &CommandExecutor{
		store:     store,
		startTime: time.Now(),
	}

	executor.commands = map[string]CommandHandler{
		"PING":    executor.ping,
		"GET":     executor.get,
		"SET":     executor.set,
		"DEL":     executor.del,
		"HSET":    executor.hset,
		"HGET":    executor.hget,
		"HGETALL": executor.hgetall,
		"HEXISTS": executor.hexists,
		"HDEL":    executor.hdel,
		"FLUSHDB": executor.flushdb,
		"INFO":    executor.info,
		"EXPIRE":  executor.expire,
		"TTL":     executor.ttl,
		"COMMAND": executor.command,
	}

	return executor
}

func (e *CommandExecutor) Execute(cmd resp.Value) resp.Value {
	if cmd.Typ != "array" || len(cmd.Array) == 0 {
		return resp.Value{Typ: "error", Str: "Invalid command format"}
	}

	command := strings.ToUpper(cmd.Array[0].Bulk)
	args := cmd.Array[1:]

	if handler, exists := e.commands[command]; exists {
		return handler(args)
	}

	return resp.Value{Typ: "error", Str: "Unknown command '" + command + "'"}
}

func (e *CommandExecutor) RegisterCommand(name string, handler CommandHandler) {
	e.commands[strings.ToUpper(name)] = handler
}

// ping обработчик команды PING
func (e *CommandExecutor) ping(args []resp.Value) resp.Value {
	if len(args) == 0 {
		return resp.Value{Typ: "string", Str: "PONG"}
	}
	return resp.Value{Typ: "string", Str: args[0].Bulk}
}

func (e *CommandExecutor) get(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'GET' command"}
	}

	value, found := e.store.Get(args[0].Bulk)
	if !found {
		return resp.Value{Typ: "null"}
	}

	return resp.Value{Typ: "bulk", Bulk: value}
}

func (e *CommandExecutor) set(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'SET' command"}
	}

	key := args[0].Bulk
	value := args[1].Bulk
	var ttl time.Duration

	// опции (NX, XX, EX, PX)
	if len(args) > 2 {
		for i := 2; i < len(args); i++ {
			arg := strings.ToUpper(args[i].Bulk)
			switch arg {
			case "NX":
				if _, exists := e.store.Get(key); exists {
					return resp.Value{Typ: "null"}
				}
			case "XX":
				if _, exists := e.store.Get(key); !exists {
					return resp.Value{Typ: "null"}
				}
			case "EX":
				if i+1 >= len(args) {
					return resp.Value{Typ: "error", Str: "ERR syntax error"}
				}
				seconds, err := strconv.Atoi(args[i+1].Bulk)
				if err != nil {
					return resp.Value{Typ: "error", Str: "ERR invalid expire time"}
				}
				ttl = time.Duration(seconds) * time.Second
				i++
			case "PX":
				if i+1 >= len(args) {
					return resp.Value{Typ: "error", Str: "ERR syntax error"}
				}
				millis, err := strconv.Atoi(args[i+1].Bulk)
				if err != nil {
					return resp.Value{Typ: "error", Str: "ERR invalid expire time"}
				}
				ttl = time.Duration(millis) * time.Millisecond
				i++
			}
		}
	}

	e.store.Set(key, value, ttl)
	return resp.Value{Typ: "string", Str: "OK"}
}

func (e *CommandExecutor) del(args []resp.Value) resp.Value {
	if len(args) < 1 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'DEL' command"}
	}

	deleted := 0
	for _, arg := range args {
		if err := e.store.Delete(arg.Bulk); err == nil {
			deleted++
		}
	}

	return resp.Value{Typ: "integer", Num: deleted}
}

func (e *CommandExecutor) hset(args []resp.Value) resp.Value {
	if len(args) < 3 || len(args)%2 != 1 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'HSET' command"}
	}

	hash := args[0].Bulk
	fields := 0
	var ttl time.Duration = 0

	// Проверяем, есть ли TTL (последний аргумент должен быть числом)
	if len(args) >= 4 {
		if ttlSec, err := strconv.Atoi(args[len(args)-1].Bulk); err == nil {
			ttl = time.Duration(ttlSec) * time.Second
			args = args[:len(args)-1] // удаляем TTL из аргументов
		}
	}

	for i := 1; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for field-value pairs"}
		}
		field := args[i].Bulk
		value := args[i+1].Bulk
		e.store.HSet(hash, field, value, ttl)
		fields++
	}

	return resp.Value{Typ: "integer", Num: fields}
}

func (e *CommandExecutor) hget(args []resp.Value) resp.Value {
	if len(args) != 2 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'HGET' command"}
	}

	value, found := e.store.HGet(args[0].Bulk, args[1].Bulk)
	if !found {
		return resp.Value{Typ: "null"}
	}

	return resp.Value{Typ: "bulk", Bulk: value}
}

func (e *CommandExecutor) hdel(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'HDEL' command"}
	}

	hash := args[0].Bulk
	deleted := 0

	for i := 1; i < len(args); i++ {
		if err := e.store.HDelete(hash, args[i].Bulk); err == nil {
			deleted++
		}
	}

	return resp.Value{Typ: "integer", Num: deleted}
}

func (e *CommandExecutor) hgetall(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'HGETALL' command"}
	}

	collection := args[0].Bulk
	fields := e.store.HGetAll(collection)

	if fields == nil {
		// Коллекции нет → пустой массив
		return resp.Value{Typ: "array", Array: []resp.Value{}}
	}

	var result []resp.Value
	for field, value := range fields {
		result = append(result, resp.Value{Typ: "bulk", Bulk: field})
		result = append(result, resp.Value{Typ: "bulk", Bulk: value})
	}

	return resp.Value{Typ: "array", Array: result}
}

func (e *CommandExecutor) hexists(args []resp.Value) resp.Value {
	if len(args) != 2 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'HEXISTS' command"}
	}

	collection := args[0].Bulk
	field := args[1].Bulk

	if e.store.HExists(collection, field) {
		return resp.Value{Typ: "integer", Num: 1}
	}

	return resp.Value{Typ: "integer", Num: 0}
}

func (e *CommandExecutor) flushdb(args []resp.Value) resp.Value {
	if len(args) > 0 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'FLUSHDB' command"}
	}

	e.store.FlushDB()
	return resp.Value{Typ: "string", Str: "OK"}
}

func (e *CommandExecutor) hlen(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'HLEN' command"}
	}

	collection := args[0].Bulk
	return resp.Value{Typ: "integer", Num: e.store.HLen(collection)}
}

func (e *CommandExecutor) expire(args []resp.Value) resp.Value {
	if len(args) != 2 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'EXPIRE' command"}
	}

	seconds, err := strconv.Atoi(args[1].Bulk)
	if err != nil {
		return resp.Value{Typ: "error", Str: "ERR invalid expire time"}
	}

	key := args[0].Bulk
	ttl := time.Duration(seconds) * time.Second

	err = e.store.SetTTL(key, ttl)
	if err != nil {
		return resp.Value{Typ: "integer", Num: 0}
	}

	return resp.Value{Typ: "integer", Num: 1}
}

func (e *CommandExecutor) ttl(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for 'TTL' command"}
	}

	key := args[0].Bulk

	_, found := e.store.Get(key)
	if !found {
		return resp.Value{Typ: "integer", Num: -2}
	}

	remaining, err := e.store.TTL(key)
	if err != nil {
		if err.Error() == "key has no TTL" {
			return resp.Value{Typ: "integer", Num: -1} // нет TTL
		}
		// Если ключ просрочен, но ещё не удалён
		return resp.Value{Typ: "integer", Num: -2}
	}

	return resp.Value{Typ: "integer", Num: int(remaining.Seconds())}
}

func (e *CommandExecutor) info(args []resp.Value) resp.Value {
	uptime := time.Since(e.startTime).Round(time.Second)
	info := map[string]string{
		"server":      "keyvalue",
		"version":     "1.0.0",
		"uptime":      uptime.String(),
		"uptime_secs": strconv.Itoa(int(uptime.Seconds())),
	}

	var sections []string
	if len(args) > 0 {
		sections = strings.Split(strings.ToLower(args[0].Bulk), ",")
	}

	var result strings.Builder
	for section, value := range info {
		if len(sections) == 0 || contains(sections, section) {
			result.WriteString(section)
			result.WriteString(":")
			result.WriteString(value)
			result.WriteString("\r\n")
		}
	}

	return resp.Value{Typ: "bulk", Bulk: result.String()}
}

func (e *CommandExecutor) command(args []resp.Value) resp.Value {
	commands := []string{"PING", "GET", "SET", "DEL",
		"HSET", "HGET", "HGETALL", "HEXISTS", "HDEL",
		"FLUSHDB", "INFO", "EXPIRE", "TTL", "COMMAND"}
	return resp.Value{Typ: "array", Array: toRespArray(commands)}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func toRespArray(items []string) []resp.Value {
	result := make([]resp.Value, len(items))
	for i, item := range items {
		result[i] = resp.Value{Typ: "bulk", Bulk: item}
	}
	return result
}
