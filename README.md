# GophKeeper

Менеджер паролей и секретов с клиент-сайд шифрованием (Zero Trust).

GophKeeper — клиент-серверное приложение для безопасного хранения паролей, текстовых заметок, банковских карт и бинарных данных. Все данные шифруются на стороне клиента AES-256-GCM — сервер никогда не имеет доступа к незашифрованным данным.

## Возможности

- 4 типа данных: логин/пароль, текст, банковская карта, бинарные файлы
- Zero Trust: сервер не видит содержимое секретов
- AES-256-GCM с уникальным nonce на каждый секрет
- Мастер-ключ через Argon2id (защита от брутфорса)
- JWT + bcrypt для аутентификации
- PostgreSQL в production, in-memory для разработки (без БД)
- Кроссплатформенный: Windows, Linux, macOS

## Быстрый старт

### 1. Запуск сервера (без базы данных)

```powershell
# in-memory режим — PostgreSQL не нужен
$env:JWT_SECRET="my-secret-key"
go run ./cmd/server
```

Сервер запустится на http://0.0.0.0:8080.

### 2. Сборка клиента

```powershell
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=$(date -u +%Y-%m-%d) -X main.commitHash=abc1234" -o gophkeeper.exe ./cmd/client
```

### 3. Регистрация и работа

```powershell
# Регистрация
.\gophkeeper.exe register user@example.com

# Вход
.\gophkeeper.exe login user@example.com

# Сохранить пароль от GitHub
.\gophkeeper.exe add credential --title "GitHub" --login "user" --password "pass123"

# Посмотреть список
.\gophkeeper.exe get --list

# Открыть секрет
.\gophkeeper.exe get <uuid-секрета>

# Информация о версии
.\gophkeeper.exe version
```

## CLI команды

Команды клиента:

- register — регистрация. Пример: `gophkeeper register user@example.com`
- login — вход в систему. Пример: `gophkeeper login user@example.com`
- add credential — сохранить логин и пароль. Пример: `gophkeeper add credential --title "Git" --login "user" --password "pass"`
- add text — текстовая заметка. Пример: `gophkeeper add text --title "Note" --content "Hello"`
- add card — банковская карта. Пример: `gophkeeper add card --title "Visa" --number "1234" --holder "CARD HOLDER" --cvv "123" --expires "12/28"`
- get --list — список всех секретов
- get <id> — детали секрета по UUID
- sync — синхронизация с сервером
- version — версия и сборка

Глобальный флаг: `-s, --server` — адрес сервера (host:port).

## Типы данных

- credential — логин и пароль (`login`, `password`)
- text — произвольный текст (`content`)
- card — банковская карта (`number`, `holder`, `cvv`, `expires`)
- binary — бинарные данные (`filename`, `content` в base64)

## Конфигурация сервера

Приоритет: переменные окружения > JSON-файл > значения по умолчанию.

Переменные окружения:

- SERVER_HOST — хост (по умолчанию 0.0.0.0)
- SERVER_PORT — порт (по умолчанию 8080)
- DATABASE_DSN — PostgreSQL DSN (пусто = in-memory)
- JWT_SECRET — секрет JWT (обязательный)
- JWT_EXPIRATION — время жизни токена (по умолчанию 24h)
- LOG_LEVEL — уровень логирования (debug, info, warn, error)

Пример с PostgreSQL:

```powershell
$env:DATABASE_DSN="postgres://user:pass@localhost:5432/gophkeeper?sslmode=disable"
$env:JWT_SECRET="my-secret-key"
go run ./cmd/server
```

Или через JSON-файл:

```powershell
$env:CONFIG_PATH="docs/server.json"
$env:JWT_SECRET="my-secret-key"
go run ./cmd/server
```

## Сборка под разные платформы

Windows:

```powershell
$env:GOOS="windows"; $env:GOARCH="amd64"
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2026-07-12 -X main.commitHash=abc1234" -o gophkeeper.exe ./cmd/client
```

Linux:

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2026-07-12 -X main.commitHash=abc1234" -o gophkeeper-linux ./cmd/client
```

macOS (Intel):

```powershell
$env:GOOS="darwin"; $env:GOARCH="amd64"
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2026-07-12 -X main.commitHash=abc1234" -o gophkeeper-darwin ./cmd/client
```

macOS (Apple Silicon):

```powershell
$env:GOOS="darwin"; $env:GOARCH="arm64"
go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2026-07-12 -X main.commitHash=abc1234" -o gophkeeper-darwin-arm64 ./cmd/client
```

## Запуск тестов

```powershell
# Все тесты
go test ./...

# С coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Структура проекта

Папки и их назначение:

- cmd/client — точка входа CLI клиента
- cmd/server — точка входа сервера
- docs — инструкция по сборке (BUILD.md) и пример конфига сервера (server.json)
- migrations — SQL-миграции для PostgreSQL
- internal/auth — JWT и bcrypt
- internal/client — CLI клиент (команды, конфиг, HTTP клиент, сервис шифрования)
- internal/crypto — AES-256-GCM и Argon2id KDF
- internal/domain — модели данных, интерфейсы, ошибки
- internal/pkg — логер и формат ответов API
- internal/server — сервер (сборка зависимостей, конфиг, хендлеры, middleware, репозитории, сервисы)
