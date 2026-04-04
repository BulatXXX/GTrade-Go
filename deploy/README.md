# Deploy

Локальные артефакты развертывания вынесены сюда.

Файлы:

- `docker-compose.yml` — локальная docker-compose схема
- `.env.example` — шаблон переменных окружения для локального запуска

Быстрый старт:

```bash
cp deploy/.env.example deploy/.env
make up
```

`Makefile` использует `deploy/docker-compose.yml` и читает переменные из `deploy/.env`.
