# auth-service

Сервис аутентификации GTrade.

## Что уже готово

- HTTP-сервер на Gin
- загрузка конфигурации из env
- подключение к PostgreSQL через repository layer
- хранение пользователей в `users`
- хранение refresh token'ов в `refresh_tokens`
- bcrypt-хеширование паролей
- JWT access/refresh token flow

## Готовые endpoint'ы

- `GET /health`
- `POST /register`
- `POST /login`
- `POST /refresh`

## Пока не реализовано

- `POST /password/reset/request`
- `POST /password/reset/confirm`
- `POST /email/verify`

Сейчас эти endpoint'ы работают как заглушки и возвращают статус `not_implemented`.

## Smoke tests

Смоки лежат в `internal/http/router_smoke_test.go`.

Проверка:

```bash
env GOCACHE=/tmp/gocache-auth go test ./...
```
