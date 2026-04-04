# Deploy

Локальные артефакты развертывания вынесены сюда.

Файлы:

- `docker-compose.yml` — локальная docker-compose схема
- `docker-compose.auth.yml` — минимальная схема только для `postgres-auth` и `auth-service`
- `.env.example` — шаблон переменных окружения для локального запуска

Быстрый старт:

```bash
cp deploy/.env.example deploy/.env
make up
```

`Makefile` использует `deploy/docker-compose.yml` и читает переменные из `deploy/.env`.

Минимальный запуск только auth:

```bash
cp deploy/.env.example deploy/.env
make auth-up
make auth-logs
```

Минимальный запуск только notification:

```bash
cp deploy/.env.example deploy/.env
make notification-up
make notification-logs
```

Минимальный запуск только catalog:

```bash
cp deploy/.env.example deploy/.env
make catalog-up
make catalog-logs
```

Тесты `auth-service` одной командой:

```bash
cp deploy/.env.example deploy/.env
make auth-test
```

Только интеграционные тесты с реальной Postgres:

```bash
make auth-test-integration
```

Тесты `notification-service` одной командой:

```bash
make notification-test
```

Только интеграционные тесты `notification-service`:

```bash
make notification-test-integration
```

Тесты `catalog-service` одной командой:

```bash
cp deploy/.env.example deploy/.env
make catalog-test
```

Только интеграционные тесты `catalog-service`:

```bash
make catalog-test-integration
```

Backup базы `catalog-service`:

```bash
make catalog-backup
```

Restore базы `catalog-service`:

```bash
make catalog-restore BACKUP_FILE=backups/catalog-20260405-013000.dump
```

Живой e2e-контур между `auth-service` и `notification-service`:

```bash
make auth-notification-e2e-test
```
