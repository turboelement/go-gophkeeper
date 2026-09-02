# Сборка и запуск GophKeeper

Все примеры — для PowerShell (Windows 11). Для Linux/macOS команды аналогичны, отличаются только переменные окружения ($env:NAME вместо export NAME=...).

## 1. Сборка клиента

Собрать под текущую платформу:

```powershell
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2026-07-12 -X main.commitHash=abc1234" -o gophkeeper.exe ./cmd/client
```

Собрать под все платформы:

```powershell
# Windows
$env:GOOS="windows"; $env:GOARCH="amd64"
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2026-07-12 -X main.commitHash=abc1234" -o gophkeeper-windows-amd64.exe ./cmd/client

# Linux
$env:GOOS="linux"; $env:GOARCH="amd64"
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2026-07-12 -X main.commitHash=abc1234" -o gophkeeper-linux-amd64 ./cmd/client

# macOS (Intel)
$env:GOOS="darwin"; $env:GOARCH="amd64"
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2026-07-12 -X main.commitHash=abc1234" -o gophkeeper-darwin-amd64 ./cmd/client

# macOS (Apple Silicon)
$env:GOOS="darwin"; $env:GOARCH="arm64"
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2026-07-12 -X main.commitHash=abc1234" -o gophkeeper-darwin-arm64 ./cmd/client
```

## 2. Сборка сервера

```powershell
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2026-07-12" -o gophkeeper-server.exe ./cmd/server
```

## 3. Запуск сервера (in-memory, без PostgreSQL)

Данные хранятся в памяти. При остановке сервера всё пропадает. Подходит для тестирования.

```powershell
$env:JWT_SECRET="my-secret-key"
go run ./cmd/server
```

Или с собранным бинарём:

```powershell
$env:JWT_SECRET="my-secret-key"
.\gophkeeper-server.exe
```

Сервер слушает http://0.0.0.0:8080.

## 4. Запуск сервера с PostgreSQL

Сервер автоматически применяет миграции при старте.

```powershell
$env:DATABASE_DSN="postgres://user:password@localhost:5432/gophkeeper?sslmode=disable"
$env:JWT_SECRET="my-secret-key"
go run ./cmd/server
```

## 5. Работа с CLI клиентом

Регистрация нового пользователя:

```powershell
.\gophkeeper.exe register user@example.com
```

Вход в систему:

```powershell
.\gophkeeper.exe login user@example.com
```

Сохранить пароль от сайта:

```powershell
.\gophkeeper.exe add credential --title "GitHub" --login "ivanov" --password "qwerty123"
```

Текстовая заметка:

```powershell
.\gophkeeper.exe add text --title "Важная заметка" --content "Купить молоко"
```

Банковская карта:

```powershell
.\gophkeeper.exe add card --title "Зарплатная" --number "4111111111111111" --holder "IVAN IVANOV" --cvv "123" --expires "12/28"
```

Список всех секретов:

```powershell
.\gophkeeper.exe get --list
```

Детали конкретного секрета:

```powershell
.\gophkeeper.exe get 550e8400-e29b-41d4-a716-446655440001
```

Синхронизация с сервером:

```powershell
.\gophkeeper.exe sync
```

Версия и сборка:

```powershell
.\gophkeeper.exe version
```

Указать сервер вручную:

```powershell
.\gophkeeper.exe -s myserver.com:8080 login user@example.com
.\gophkeeper.exe -s 192.168.1.100:8080 add credential --title "Test" --login "u" --password "p"
```

## 6. Конфигурация через JSON

Сервер можно настроить через JSON-файл. Пример в docs/server.json:

```powershell
$env:CONFIG_PATH="docs/server.json"
$env:JWT_SECRET="my-secret-key"
go run ./cmd/server
```

Приоритет значений: переменные окружения > JSON-файл > значения по умолчанию.

## 7. Переменные окружения

Сервер:

- SERVER_HOST — хост (по умолчанию 0.0.0.0)
- SERVER_PORT — порт (по умолчанию 8080)
- SERVER_READ_TIMEOUT — таймаут чтения HTTP (по умолчанию 10s)
- SERVER_WRITE_TIMEOUT — таймаут записи HTTP (по умолчанию 10s)
- DATABASE_DSN — PostgreSQL DSN (пусто = in-memory)
- JWT_SECRET — секрет для JWT (обязательный)
- JWT_EXPIRATION — время жизни токена (по умолчанию 24h)
- LOG_LEVEL — уровень логирования: debug, info, warn, error (по умолчанию info)
- LOG_DEV_MODE — true для цветного вывода (по умолчанию false)
- CONFIG_PATH — путь к JSON-конфигу

Клиент:

- GOPHKEEPER_SERVER — адрес сервера (по умолчанию localhost:8080)

## 8. Тестирование

Все тесты:

```powershell
go test ./...
```

С детализацией:

```powershell
go test ./... -v
```

Coverage в консоли:

```powershell
go test -cover ./internal/auth/ ./internal/crypto/ ./internal/client/service/ ./internal/server/repository/
```

Coverage с HTML-отчётом:

```powershell
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```
