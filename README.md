# key-value-storage

Это упрощённая реализация Redis на Go, поддерживающая базовые команды, хранение данных в памяти и работу с ключами. Проект разработан для изучения принципов работы key-value баз данных.

## Особенности

- **Поддержка RESP (REdis Serialization Protocol)**
- **Основные структуры данных**:
  - Строки (`SET`, `GET`, `DEL`)
  - Хеши (`HSET`, `HGET`, `HDEL`)
- **Механизм AOF (Append Only File)**:
  - Журналирование всех операций изменения
  - Восстановление состояния при перезапуске
  - Настраиваемая политика синхронизации
- **Поддержка TTL**:
  - `EXPIRE`, `TTL` команды
  - Автоматическое удаление просроченных ключей

```bash
git clone https://github.com/yourusername/keyvalue-server.git
cd keyvalue-server
go build -o kv-server cmd/server/main.go
```

```Docker
docker build -t keyvalue-server .
docker run -d -p 6379:6379 -v ./data:/data --name kv-server keyvalue-server
```