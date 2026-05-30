# VideoSaver

Telegram inline-бот для скачивания видео из социальных сетей.

> Статус: переписывается с Python (Playwright + savefrom) на Go (telebot.v3) + Python insta-resolver (instaloader). Этот коммит — foundation skeleton.

## Архитектура

- `bot/` — Go-сервис (Telegram-бот, оркестрация, кэш, extractors)
- `insta-resolver/` — Python-сервис, обёртка над instaloader (только для Instagram)
- Redis — кэш + job-queue + account pool state

См. `docs/superpowers/specs/2026-05-30-videosaver-go-migration-design.md` — полный design.
См. `docs/superpowers/plans/` — пошаговые планы реализации.

## Локальный запуск

Требуется Docker и Docker Compose v2.

1. Скопируй конфиг и пропиши токен бота:

   ```bash
   cp .env.example .env
   # отредактируй .env, поставь BOT_TOKEN от @BotFather
   ```

2. Подними всё:

   ```bash
   docker compose up -d --build
   ```

3. Проверь:

   ```bash
   docker compose logs -f bot
   ```

   В Telegram отправь `/start` своему боту — он должен ответить заглушкой.

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `BOT_TOKEN` | — | Токен от @BotFather (обязательно) |
| `REDIS_URL` | `redis://redis:6379/0` | URL Redis |
| `INSTA_RESOLVER_URL` | `http://insta-resolver:8000` | URL Python-сервиса |
| `CACHE_TTL_SEC` | `86400` | TTL кэша (24 часа) |
| `LOG_LEVEL` | `info` | trace / debug / info / warn / error |

## Разработка

### Go-бот

```bash
cd bot
go test ./...
go build ./cmd/bot
```

### Python insta-resolver

```bash
cd insta-resolver
pip install -r requirements.txt
uvicorn app.main:app --reload
```

## Roadmap

- **v0.1** (текущая разработка): MVP с 5 платформами (Instagram, YouTube, TikTok, Twitter, Facebook), один IG-аккаунт
- **v0.2**: ещё ~10 платформ, пул IG-аккаунтов, residential proxy, audio-режим
- **v0.3**: собственная `go-instagram` библиотека вместо Python-сервиса
- **v0.4**: webhook, health-check, метрики

## Лицензия

MIT
