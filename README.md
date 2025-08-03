# key-value-storage

Это упрощённая реализация Redis на Go, поддерживающая базовые команды, хранение данных в памяти и работу с ключами. Проект разработан для изучения принципов работы key-value баз данных.

---

##  Особенности

- **Полная совместимость с RESP** — работает с `redis-cli`, `go-redis`, `redis-py` и другими клиентами.
- **In-memory хранилище** с поддержкой строк и хэшей.
- **TTL (время жизни ключей)** — автоматическое удаление устаревших данных.
- **AOF (Append-Only File)** — персистентность с возможностью восстановления после перезапуска.
- **Graceful shutdown** — безопасное завершение с сохранением данных.
- **Потокобезопасность** — корректная работа в многопоточной среде.
- **Минимализм** — нет внешних зависимостей, только стандартная библиотека Go.

---

## Архитектура

| Уровень          | Ответственность                              |
|------------------|----------------------------------------------|
| **TCP Layer**    | Принимает клиентские подключения             |
| ↓                |                                              |
| **RESP Parser**  | Парсинг и сериализация команд в формате RESP |
| ↓                |                                              |
| **Command Executor** | Маршрутизация и выполнение команд       |
| ↓                |                                              |
| **Storage Layer**| In-memory хранилище с поддержкой TTL и хэшей |
| ↓                |                                              |
| **AOF Engine**   | Персистентность: запись и восстановление AOF |

### 1. Установка

```bash
git clone https://github.com/AlexSamarskii/key-value-storage.git
cd key-value-storage/cmd/server
go build -o ../../key-value-storage
./key-value-storage
```

или

```Docker
docker build -t keyvalue-server .
docker run -d -p 6379:6379 -v ./data:/data --name kv-server keyvalue-server
```

### 2. Подключение через redis-cli

```bash
redis-cli
> PING
PONG
> SET name "Alice"
OK
> GET name
"Alice"
> EXPIRE name 60
(integer) 1
```

После перезапуска сервера все данные будут восстановлены из database.aof

## Поддерживаемые команды

| Команда         | Группа     | Описание                                      |
|-----------------|-----------|-----------------------------------------------|
| `GET key`       | Ключи     | Получить значение по ключу                    |
| `SET key value [NX\|XX] [EX seconds]` | Ключи | Установить значение с опциями             |
| `DEL key ...`   | Ключи     | Удалить один или несколько ключей            |
| `EXPIRE key seconds` | Ключи | Установить TTL (в секундах)                  |
| `TTL key`       | Ключи     | Получить оставшееся время жизни ключа         |
| `PING`          | Ключи     | Проверка связи — возвращает `PONG`           |
| `FLUSHDB`       | Ключи     | Удалить все ключи                             |
| `HSET hash field value ...` | Хэши | Установить поля в хэше                   |
| `HGET hash field` | Хэши     | Получить значение поля                       |
| `HGETALL hash`  | Хэши      | Получить все поля и значения хэша            |
| `HEXISTS hash field` | Хэши | Проверить, существует ли поле               |
| `HDEL hash field ...` | Хэши | Удалить одно или несколько полей            |
| `HDELALL hash`  | Хэши      | Удалить всю хэш-коллекцию                    |
| `INFO`          | Система   | Вывести информацию о сервере                 |
| `COMMAND`       | Система   | Получить список поддерживаемых команд        |

Примеры

```bash 
SET key value NX EX 60           # Условная установка с TTL
HGETALL user:1                   # Получить все поля хэша
INFO                             # Информация о сервере
COMMAND                          # Список поддерживаемых команд
```

## Персистентность: AOF

Каждая записывающая команда (например, SET, HSET) добавляется в AOF-файл в формате RESP. При запуске сервер читает файл и воссоздаёт состояние.

## Для разработчиков

### Добавление новой команды

```go
executor.RegisterCommand("ECHO", func(args []resp.Value) resp.Value {
    if len(args) == 0 {
        return resp.Value{Typ: "null"}
    }
    return resp.Value{Typ: "bulk", Bulk: args[0].Bulk}
})
```

### Интеграция в своё приложение

```go
import "github.com/ваш-проект/server"
srv := server.NewServer(server.Config{
    Port:        6380,
    AofFilename: "app-data.aof",
})

if err := srv.Start(); err != nil {
    log.Fatal(err)
}
defer srv.Stop()
```

## TODO

  - **Поддержка RDB-снапшотов**
  - **Unit тесты**
