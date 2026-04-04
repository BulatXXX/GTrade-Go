# notification-service docs

Документация по локальному запуску, тестированию и ручной проверке `notification-service`.

## Что нужно

- Docker Desktop
- `make`
- `curl`
- рабочий `RESEND_API_KEY` для живой отправки через Resend

## Конфиг

В `deploy/.env` должны быть заданы:

```env
NOTIFICATION_SERVICE_PORT=8085
NOTIFICATION_SERVICE_DATABASE_URL=postgres://gtrade:gtrade@postgres-notification:5432/gtrade_notification?sslmode=disable
RESEND_API_KEY=...
RESEND_FROM_EMAIL=GTrade <noreply@2dots.online>
EMAIL_PROVIDER=resend
```

Для локальной разработки без Resend можно использовать:

```env
EMAIL_PROVIDER=mock
```

## Быстрый старт

1. Подготовить env:

```bash
cp deploy/.env.example deploy/.env
```

2. Убедиться, что в `deploy/.env` заполнены `RESEND_API_KEY`, `RESEND_FROM_EMAIL` и `EMAIL_PROVIDER`

3. Поднять только `notification-service` и его PostgreSQL:

```bash
make notification-up
```

4. Посмотреть логи:

```bash
make notification-logs
```

5. Остановить:

```bash
make notification-down
```

## Полный локальный стек

Если нужен весь локальный стек проекта:

```bash
make up
```

Логи всего стека:

```bash
make logs
```

Остановить весь стек:

```bash
make down
```

## Минимальный запуск только PostgreSQL notification

Через `make`:

```bash
make notification-db-up
```

Остановить:

```bash
make notification-db-down
```

Низкоуровневый эквивалент через `docker compose`:

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d postgres-notification
```

Локальный запуск сервиса без Docker:

```bash
cd services/notification-service
DATABASE_URL='postgres://gtrade:gtrade@localhost:5437/gtrade_notification?sslmode=disable' \
SERVICE_NAME='notification-service' \
PORT='8085' \
EMAIL_PROVIDER='resend' \
RESEND_API_KEY='YOUR_RESEND_API_KEY' \
RESEND_FROM_EMAIL='GTrade <noreply@2dots.online>' \
go run ./cmd/server
```

Для `mock` provider:

```bash
cd services/notification-service
DATABASE_URL='postgres://gtrade:gtrade@localhost:5437/gtrade_notification?sslmode=disable' \
SERVICE_NAME='notification-service' \
PORT='8085' \
EMAIL_PROVIDER='mock' \
go run ./cmd/server
```

## Тесты

Все тесты сервиса:

Через `make`:

```bash
make notification-test
```

Прямой вызов:

```bash
cd services/notification-service
TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5437/gtrade_notification?sslmode=disable' \
GOCACHE=/tmp/gocache-notification \
go test ./...
```

Только HTTP интеграционные тесты с реальной PostgreSQL:

Через `make`:

```bash
make notification-test-integration
```

Прямой вызов:

```bash
cd services/notification-service
TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5437/gtrade_notification?sslmode=disable' \
GOCACHE=/tmp/gocache-notification \
go test ./internal/http -run TestSendEmailIntegration -v
```

Только unit tests сервисного слоя:

```bash
cd services/notification-service
GOCACHE=/tmp/gocache-notification \
go test ./internal/service -v
```

## Ручная проверка API

Базовый адрес:

```text
http://localhost:8085
```

### 1. Health

```bash
curl http://localhost:8085/health
```

Ожидаемый результат:

```json
{
  "status": "ok",
  "service": "notification-service"
}
```

### 2. Send email

```bash
curl -X POST http://localhost:8085/send-email \
  -H 'Content-Type: application/json' \
  -d '{"to":"konnor15000@mail.ru","subject":"ТЕСТ notification service","html_body":"<p>ТЕСТ notification service</p>","text_body":"ТЕСТ notification service"}'
```

Ожидаемый результат:

```json
{
  "id": 1,
  "status": "queued"
}
```

После этого в `notification_outbox` должна появиться запись со статусом `sent` или `failed`.

## Типовые ошибки

### Missing recipient

```bash
curl -X POST http://localhost:8085/send-email \
  -H 'Content-Type: application/json' \
  -d '{"subject":"Test","text_body":"Hello"}'
```

Ожидаемый результат:

```json
{
  "error": "to is required"
}
```

HTTP status: `400`

### Resend domain is not verified

```json
{
  "error": "resend send email failed: status=403 message=The 2dots.online domain is not verified. Please, add and verify your domain on https://resend.com/domains"
}
```

Такой ответ означает, что `RESEND_FROM_EMAIL` не совпадает с verified domain или Resend API key не видит этот домен в текущем workspace.
