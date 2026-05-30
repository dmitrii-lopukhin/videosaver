# VideoSaver

Telegram inline bot for downloading videos from social networks.

> Status: being rewritten from Python (Playwright + savefrom) to Go (telebot.v3) + a Python insta-resolver (instaloader). This commit is the foundation skeleton.

## Architecture

- `bot/` — Go service (Telegram bot, orchestration, cache, extractors)
- `insta-resolver/` — Python service, a wrapper around instaloader (Instagram only)
- Redis — cache + job queue + account pool state

## Local run

Requires Docker and Docker Compose v2.

1. Copy the config and set the bot token:

   ```bash
   cp .env.example .env
   # edit .env, set BOT_TOKEN from @BotFather
   ```

2. Bring everything up:

   ```bash
   docker compose up -d --build
   ```

3. Check it:

   ```bash
   docker compose logs -f bot
   ```

   In Telegram, send `/start` to your bot — it should reply with a placeholder.

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `BOT_TOKEN` | — | Token from @BotFather (required) |
| `REDIS_URL` | `redis://redis:6379/0` | Redis URL |
| `INSTA_RESOLVER_URL` | `http://insta-resolver:8000` | Python service URL |
| `CACHE_TTL_SEC` | `86400` | Cache TTL (24 hours) |
| `LOG_LEVEL` | `info` | trace / debug / info / warn / error |

## Development

### Go bot

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

- **v0.1** (current development): MVP with 5 platforms (Instagram, YouTube, TikTok, Twitter, Facebook), single IG account
- **v0.2**: ~10 more platforms, IG account pool, residential proxy, audio mode
- **v0.3**: an in-house `go-instagram` library instead of the Python service
- **v0.4**: webhook, health check, metrics

## License

MIT
