# notification-service

Сервис email-уведомлений GTrade.

- HTTP-сервер на отдельном порту
- `GET /health`
- `POST /send-email`
- загрузка конфигурации из env
- подключение к PostgreSQL через repository layer
- `notification_outbox` в PostgreSQL
- mock provider
- Resend provider
- smoke, unit и integration tests

См. также:

- `progress.md`
- `docs/README.md`
- `docs/openapi.yaml`
