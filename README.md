# Invoice Generator (Go + Gin)

Небольшой веб-сервис для загрузки `.xlsx` файлов заявок и формирования накладной.

## Требования

- Go `1.25+`
- Docker (опционально, если запускать в контейнере)

## Запуск локально

1. Установи зависимости:

```bash
go mod download
```

2. Запусти приложение:

```bash
go run main.go
```

По умолчанию сервер стартует на `http://localhost:8080`.

Доступные страницы:

- `http://localhost:8080/bread`
- `http://localhost:8080/kond`

## Сборка бинарника

```bash
go build -o invoice_generator ./main.go
./invoice_generator
```

## Запуск через Docker

1. Собери образ:

```bash
docker build -t invoice_generator:local .
```

2. Запусти контейнер:

```bash
mkdir -p uploads
docker run --rm -p 8080:8080 -v $(pwd)/uploads:/app/uploads invoice_generator:local
```

```bash
mkdir -p uploads
docker run -d --name invoice_generator -p 8080:8080 -v $(pwd)/uploads:/app/uploads invoice_generator:local
```

Если хост использует SELinux (например Fedora/CentOS/RHEL), добавь метку `:Z`:

```bash
docker run --rm -p 8080:8080 -v $(pwd)/uploads:/app/uploads:Z invoice_generator:local
```

## Домен без порта и HTTPS

Для доступа как `https://your-domain.com` без `:8080` используй `docker compose` с Caddy.

1. Убедись, что DNS `A` запись домена указывает на IP сервера.
2. Убедись, что на сервере открыты входящие порты `80` и `443`.
3. Собери образ приложения:

```bash
docker build -t invoice_generator:local .
```

4. Создай `.env` в корне проекта:

```bash
echo "DOMAIN=your-domain.com" > .env
```

5. Запусти сервисы:

```bash
mkdir -p uploads
docker compose up -d
```

6. Проверь:
- `https://your-domain.com/bread`
- `https://your-domain.com/kond`

Если хост использует SELinux (например Fedora/CentOS/RHEL), замени volume в `docker-compose.yml` на:

```yaml
    volumes:
      - ./uploads:/app/uploads:Z
```

## Полезно знать

- Приложение использует папку `uploads/` для временных файлов.
- В `.gitignore` содержимое `uploads/` игнорируется, но папка хранится в репозитории через `uploads/.gitkeep`.
- Порт можно изменить переменной окружения `PORT`, например:

```bash
PORT=9090 go run main.go
```
