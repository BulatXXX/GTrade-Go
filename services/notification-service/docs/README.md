# notification-service docs

Документация по локальному запуску, тестированию и ручной проверке `notification-service`.

## Что нужно

- Docker Desktop
- `make`
- `curl`
- рабочий `RESEND_API_KEY` для живой отправки через Resend

## Быстрый старт

1. Подготовить env:

```bash
cp deploy/.env.example deploy/.env
```

2. Убедиться, что в `deploy/.env` заполнены:

```env
RESEND_API_KEY=...
RESEND_FROM_EMAIL=GTrade <noreply@2dots.online>
EMAIL_PROVIDER=resend
```

3. Поднять `notification-service` и его PostgreSQL:

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

## Тесты

Все тесты сервиса:

```bash
make notification-test
```

Только интеграционные тесты с реальной PostgreSQL:

```bash
make notification-test-integration
```

Живой системный e2e-тест связки `auth-service -> notification-service`:

```bash
make auth-notification-e2e-test
```

Этот сценарий поднимает реальные контейнеры `auth-service`, `notification-service`, `postgres-auth`, `postgres-notification` и проверяет:

- `POST /password/reset/request`
- `POST /email/verify`
- отсутствие токенов в публичном API `auth-service`
- появление записей в `notification_outbox`

Для стабильности этот e2e-контур использует `EMAIL_PROVIDER=mock`, а не реальный Resend.

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

Если нужен repeatable системный тест без реальной отправки письма:

```bash
make auth-notification-e2e-test
```

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

### Missing subject

```bash
curl -X POST http://localhost:8085/send-email \
  -H 'Content-Type: application/json' \
  -d '{"to":"user@example.com","text_body":"Hello"}'
```

Ожидаемый результат:

```json
{
  "error": "subject is required"
}
```

HTTP status: `400`

### Missing body

```bash
curl -X POST http://localhost:8085/send-email \
  -H 'Content-Type: application/json' \
  -d '{"to":"user@example.com","subject":"Test"}'
```

Ожидаемый результат:

```json
{
  "error": "html_body or text_body is required"
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
