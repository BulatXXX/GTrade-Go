# notification-service

Сервис уведомлений GTrade.

## Что уже готово

- HTTP-сервер на Gin
- загрузка конфигурации из env
- подключение к PostgreSQL через repository layer
- endpoint `GET /health`
- endpoint `POST /send-email`
- валидация payload для отправки email
- `notification_outbox` в PostgreSQL
- запись исходящих писем в outbox перед отправкой
- обновление статусов outbox в `sent` и `failed`
- сохранение `provider_message_id` и текста ошибки провайдера
- абстракция email provider
- рабочий `mock` provider
- рабочая интеграция с Resend
- smoke tests HTTP-слоя
- unit tests сервисного слоя
- интеграционные тесты с реальной PostgreSQL
- живая проверка отправки через Resend

## Готовые endpoint'ы

- `GET /health`
- `POST /send-email`

## Текущий flow

- `POST /send-email` принимает `to`, `subject`, `html_body`, `text_body`
- сервис валидирует payload
- сервис создает запись в `notification_outbox` со статусом `queued`
- сервис выбирает provider (`mock` или `resend`)
- при успешной отправке запись обновляется в `sent`
- при ошибке отправки запись обновляется в `failed` и сохраняет `error_message`

## Что нужно доделать

- интеграция `auth-service` с `notification-service`
- шаблоны писем под password reset и email verification
- внутренний auth/allowlist для межсервисного вызова `POST /send-email`
- более явное разделение provider errors и validation errors на уровне HTTP
- rate limiting или защита от злоупотребления публичным email endpoint при необходимости
- отдельный docker-compose сценарий только для `notification-service` и его PostgreSQL
- swagger-ui или другой способ локально просматривать OpenAPI

## Ключевые файлы

- `internal/service/email.go` — основная бизнес-логика отправки письма и работы с outbox
- `internal/repository/postgres.go` — PostgreSQL repository для `notification_outbox`
- `internal/handler/notification.go` — HTTP handler отправки письма
- `internal/model/model.go` — request/response DTO
- `internal/http/service_routes.go` — маршруты сервиса
- `internal/service/provider/mock.go` — mock provider
- `internal/service/provider/resend.go` — интеграция с Resend API
- `migrations/0001_init.sql` — схема `notification_outbox`
- `docs/README.md` — запуск, ручная проверка и тесты
- `docs/openapi.yaml` — актуальный OpenAPI/Swagger контракт

## Следующий шаг

- подключить `auth-service` к `notification-service`
- перестать возвращать reset/verification token в API-ответах `auth-service`
- перевести auth flow на email delivery через notification layer

## Текущие ограничения

- endpoint `POST /send-email` пока открыт и не ограничен межсервисной аутентификацией
- нет фонового worker/retry механизма, отправка выполняется синхронно в HTTP request flow
- нет отдельного шаблонизатора писем, тело письма приходит готовым в запросе
- успешная отправка через Resend требует verified domain и корректный `RESEND_FROM_EMAIL`

## Тесты

- HTTP smoke tests лежат в `internal/http/router_smoke_test.go`
- интеграционные HTTP tests с PostgreSQL лежат в `internal/http/router_integration_test.go`
- unit tests сервисного слоя лежат в `internal/service/email_test.go`

Проверка:

```bash
cd services/notification-service
TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5437/gtrade_notification?sslmode=disable' GOCACHE=/tmp/gocache-notification go test ./...
```
