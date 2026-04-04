# auth-service docs

Документация по локальному запуску, тестированию и ручной проверке `auth-service`.

## Что нужно

- Docker Desktop
- `make`
- `curl`

## Быстрый старт

1. Подготовить env:

```bash
cp deploy/.env.example deploy/.env
```

2. Поднять `auth-service` и его PostgreSQL:

```bash
make auth-up
```

3. Посмотреть логи:

```bash
make auth-logs
```

4. Остановить:

```bash
make auth-down
```

## Тесты

Все тесты сервиса:

```bash
make auth-test
```

Только интеграционные тесты с реальной PostgreSQL:

```bash
make auth-test-integration
```

## Ручная проверка API

Базовый адрес:

```text
http://localhost:8081
```

### 1. Health

```bash
curl http://localhost:8081/health
```

Ожидаемый результат:

```json
{
  "status": "ok",
  "service": "auth-service"
}
```

### 2. Register

```bash
curl -X POST http://localhost:8081/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret123"}'
```

Ожидаемый результат:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### 3. Login

```bash
curl -X POST http://localhost:8081/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret123"}'
```

Ожидаемый результат:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### 4. Refresh

Подставить `refresh_token`, полученный на шаге register или login:

```bash
curl -X POST http://localhost:8081/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"PASTE_REFRESH_TOKEN_HERE"}'
```

Ожидаемый результат:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### 5. Request password reset

```bash
curl -X POST http://localhost:8081/password/reset/request \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com"}'
```

Ожидаемый результат:

```json
{
  "status": "accepted",
  "reset_token": "..."
}
```

Сейчас `reset_token` возвращается в ответе специально для dev/MVP.

### 6. Confirm password reset

Подставить `reset_token` из предыдущего шага:

```bash
curl -X POST http://localhost:8081/password/reset/confirm \
  -H 'Content-Type: application/json' \
  -d '{"token":"PASTE_RESET_TOKEN_HERE","new_password":"secret456"}'
```

Ожидаемый результат:

```json
{
  "status": "password_reset"
}
```

После этого логин со старым паролем должен перестать работать, а с новым начать работать.

### 7. Request email verification token

```bash
curl -X POST http://localhost:8081/email/verify \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com"}'
```

Ожидаемый результат:

```json
{
  "status": "verification_requested",
  "verification_token": "..."
}
```

Сейчас `verification_token` возвращается в ответе специально для dev/MVP.

### 8. Confirm email verification

Подставить `verification_token` из предыдущего шага:

```bash
curl -X POST http://localhost:8081/email/verify \
  -H 'Content-Type: application/json' \
  -d '{"token":"PASTE_VERIFICATION_TOKEN_HERE"}'
```

Ожидаемый результат:

```json
{
  "status": "verified"
}
```

## Типовые ошибки

### Duplicate register

```bash
curl -X POST http://localhost:8081/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"secret123"}'
```

Ожидаемый результат:

```json
{
  "error": "email already exists"
}
```

HTTP status: `409`

### Invalid login

```bash
curl -X POST http://localhost:8081/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"wrong"}'
```

Ожидаемый результат:

```json
{
  "error": "invalid credentials"
}
```

HTTP status: `401`

### Invalid refresh token

```bash
curl -X POST http://localhost:8081/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"bad-token"}'
```

Ожидаемый результат:

```json
{
  "error": "invalid refresh token"
}
```

HTTP status: `401`

## Swagger / OpenAPI

Текущая OpenAPI-схема лежит в:

- `services/auth-service/docs/openapi.yaml`

Это актуальный конфиг для текущего MVP auth flow.
