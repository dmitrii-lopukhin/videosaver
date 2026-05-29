# VideoSaver Bot

Telegram INLINE-бот для быстрого сохранения и отправки видео из популярных социальных сетей (TikTok, Instagram, YouTube и др.).

## Особенности

- **Inline режим**: Работает в любом чате без добавления бота в группу
- **Быстрый ответ**: Кэширование результатов в Redis для мгновенной выдачи
- **Дедупликация**: Защита от одновременных запросов на один URL
- **Модульная архитектура**: Легко добавлять новые провайдеры (SaveFrom, yt-dlp, Cobalt)
- **Production-ready**: Готов к развертыванию с Docker

## Архитектура

```
app/
├── main.py              # Точка входа приложения
├── config.py            # Конфигурация из переменных окружения
├── handlers/
│   ├── inline.py        # Обработка inline-запросов
│   └── pm.py            # Обработка личных сообщений (/start, /settings)
├── providers/
│   ├── base.py          # Базовый интерфейс провайдера
│   └── savefrom.py      # SaveFrom.net провайдер (заглушка)
└── services/
    ├── cache.py         # Redis кэш с блокировками
    ├── normalize.py     # Нормализация URL
    └── settings.py      # Пользовательские настройки
```

## Требования

- Python 3.11+
- Redis
- Telegram Bot Token
- Playwright browsers (устанавливаются автоматически)

## Локальная разработка

### 1. Клонирование и настройка

```bash
git clone <repository-url>
cd videosaver
```

### 2. Создание файла `.env`

Создайте файл `.env` в корне проекта:

```env
BOT_TOKEN=your_telegram_bot_token_here
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
# REDIS_PASSWORD=  # Опционально
```

### 3. Установка зависимостей

```bash
pip install -r requirements.txt
```

### 4. Установка браузеров Playwright

После установки зависимостей необходимо установить браузеры Playwright:

```bash
playwright install chromium
playwright install-deps chromium
```

**Примечание**: На Windows может потребоваться запуск PowerShell от имени администратора для установки системных зависимостей.

### 5. Запуск Redis (локально)

```bash
# Используя Docker Compose для разработки
docker-compose -f docker-compose.dev.yml up -d redis

# Или установите Redis локально и запустите
redis-server
```

### 6. Запуск бота

```bash
python -m app.main
```

## Запуск с Docker (локальная разработка)

### Быстрый старт

```bash
# Сборка и запуск (с hot-reload - изменения в app/ применяются автоматически)
docker-compose -f docker-compose.dev.yml up --build

# Запуск в фоне
docker-compose -f docker-compose.dev.yml up -d --build

# Просмотр логов
docker-compose -f docker-compose.dev.yml logs -f bot

# Остановка
docker-compose -f docker-compose.dev.yml down

# Остановка с удалением volumes
docker-compose -f docker-compose.dev.yml down -v
```

### Особенности dev-режима

- **Hot-reload**: Изменения в `app/` применяются автоматически без пересборки
- **Порты**: Redis доступен на `localhost:6379` для отладки
- **Volumes**: Код монтируется как volume для быстрой разработки

## Production развертывание на сервере

### 1. Подготовка

Создайте файл `.env` на сервере с production настройками:

```env
BOT_TOKEN=your_production_bot_token
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0
REDIS_PASSWORD=your_secure_password
```

### 2. Запуск с Docker Compose

```bash
# Сборка и запуск в фоне
docker-compose up -d --build

# Проверка статуса
docker-compose ps

# Просмотр логов
docker-compose logs -f bot

# Просмотр логов Redis
docker-compose logs -f redis
```

### 3. Управление

```bash
# Перезапуск бота
docker-compose restart bot

# Остановка
docker-compose down

# Остановка с удалением volumes (⚠️ удалит данные Redis)
docker-compose down -v
```

### Особенности production-режима

- **Restart policy**: `restart: always` - автоматический перезапуск при сбоях
- **Health checks**: Проверка здоровья сервисов
- **Volumes**: Только данные Redis, код в образе
- **Без hot-reload**: Изменения требуют пересборки образа

## Переход на Webhook (будущее)

Код спроектирован для легкого перехода на webhook. В `app/config.py` уже есть настройки:

```python
WEBHOOK_URL=https://yourdomain.com
WEBHOOK_PATH=/webhook
WEBHOOK_PORT=8000
```

Для перехода на webhook нужно будет:
1. Заменить `dp.start_polling()` на `dp.start_webhook()` в `app/main.py`
2. Настроить SSL сертификат
3. Настроить reverse proxy (nginx)

## Использование

1. Найдите вашего бота в Telegram
2. В любом чате напишите `@YourBotName <url>`
3. Выберите результат из списка
4. Видео будет отправлено в чат

### Примеры

```
@YourBotName https://www.tiktok.com/@user/video/1234567890
@YourBotName https://www.instagram.com/p/ABC123/
@YourBotName https://www.youtube.com/watch?v=dQw4w9WgXcQ
```

## Конфигурация

### Переменные окружения

- `BOT_TOKEN` - токен Telegram бота (обязательно)
- `REDIS_HOST` - хост Redis (по умолчанию: localhost)
- `REDIS_PORT` - порт Redis (по умолчанию: 6379)
- `REDIS_DB` - номер базы данных Redis (по умолчанию: 0)
- `REDIS_PASSWORD` - пароль Redis (опционально)
- `CACHE_TTL` - время жизни кэша в секундах (по умолчанию: 3600)
- `CACHE_LOCK_TTL` - время жизни блокировки в секундах (по умолчанию: 30)
- `INLINE_QUERY_TIMEOUT` - таймаут для inline-запроса в секундах (по умолчанию: 2.0)
- `PROVIDER_TIMEOUT_SEC` - общий таймаут для провайдера в секундах (по умолчанию: 1.5)
- `PROVIDER_TIMEOUT_MS` - таймаут для операций Playwright в миллисекундах (по умолчанию: 1500)
- `PROVIDER_MAX_CONCURRENT` - максимальное количество одновременных запросов к провайдеру (по умолчанию: 2)

### Пользовательские настройки

Пользовательские настройки хранятся в Redis по ключу `user:{user_id}:settings`:

- `mode` - режим работы: `video` | `audio` (по умолчанию: `video`)
- `confirm` - требовать подтверждение перед отправкой: `true` | `false` (по умолчанию: `false`)
- `quality` - качество видео: `best` | `720` | `480` (по умолчанию: `best`)

Настройки доступны через команду `/settings` в личных сообщениях (UI в разработке).

## Разработка провайдеров

Для добавления нового провайдера:

1. Создайте класс, наследующий `VideoProvider` из `app/providers/base.py`
2. Реализуйте методы `can_handle()` и `resolve()`
3. Добавьте провайдер в список обработчиков в `app/handlers/inline.py`

Пример:

```python
from app.providers.base import VideoProvider

class MyProvider(VideoProvider):
    def can_handle(self, url: str) -> bool:
        return "example.com" in url
    
    async def resolve(self, url: str) -> Optional[str]:
        # Ваша логика резолвинга URL
        return direct_video_url
```

## Структура проекта

- `app/main.py` - точка входа, инициализация бота и диспетчера
- `app/handlers/inline.py` - обработка inline-запросов с кэшированием
- `app/handlers/pm.py` - обработка команд в личных сообщениях
- `app/providers/base.py` - базовый интерфейс для провайдеров
- `app/providers/savefrom.py` - SaveFrom.net провайдер (заглушка)
- `app/services/cache.py` - Redis кэш с блокировками для дедупликации
- `app/services/normalize.py` - нормализация URL для консистентного кэширования
- `app/config.py` - загрузка конфигурации из переменных окружения

## Лицензия

MIT
