# tg-reminder-bot

Персональный Telegram-бот для напоминаний на Go + PostgreSQL.
Разработан под мой личный рабочий процесс — формат взаимодействия и стиль команд
настроены под личные предпочтения и могут быть неудобны другим пользователям без доработки.

## Важное замечание

Бот разработан исключительно под личное использование.
Формат ввода, язык команд и логика взаимодействия отражают мои личные предпочтения.
Если хочешь адаптировать бота под себя — начни с парсера ([internal/parser](internal/parser))
и шаблонов сообщений ([internal/bot/handler.go](internal/bot/handler.go)).

## О проекте

Бот принимает сообщения в свободной форме с описанием задачи и дедлайна,
парсит их, сохраняет напоминание и отправляет уведомление в нужное время.
Без сторонних планировщиков. Минимум внешних зависимостей.

## Функционал

- Ввод на естественном языке: `завтра в 14:00 сделать дз`
- Форматы дат: `сегодня`, `завтра`, `послезавтра`, `13 мая`
- Повторяющиеся напоминания: `каждый день в 08:00 зарядка`, `каждый понедельник в 10:00 стендап`
- Управление задачами через команды
- Whitelist по chat_id — бот отвечает только с настроенного аккаунта
- Опциональный прокси через Cloudflare Worker — для обхода блокировки Telegram из РФ
- Graceful shutdown, структурированные логи через `slog`

## Команды

| Ввод | Действие |
|---|---|
| `завтра в 14:00 сделать дз` | создать напоминание |
| `каждый понедельник в 10:00 стендап` | создать повторяющееся напоминание |
| `/list` | список активных напоминаний |
| `/cancel 3` | отменить напоминание #3 |
| `выполнил 7` | закрыть напоминание #7 |
| `/history` | последние 10 выполненных или отменённых |

## Стек

- Go 1.25+
- PostgreSQL — хранение данных
- [go-telegram-bot-api/v5](https://github.com/go-telegram-bot-api/telegram-bot-api) — интеграция с Telegram
- [golang-migrate](https://github.com/golang-migrate/migrate) — миграции базы данных
- Docker + docker-compose — локальное окружение
- Cloudflare Worker (опционально) — прокси для `api.telegram.org`

## Архитектура

```
handler → service → storage → postgres
            ↑
        scheduler (тикер 30 сек → service.ProcessDue)
            ↓
         sender → telegram (через прокси при необходимости)
```

Слоистая архитектура с явным разделением ответственности.
Storage и Sender реализованы через интерфейсы.

## Структура проекта

```
tg-reminder-bot/
├── cmd/bot/          # точка входа, wiring зависимостей
├── internal/
│   ├── bot/          # handler, sender, middleware
│   ├── parser/       # парсинг естественного языка
│   ├── scheduler/    # поллинг просроченных напоминаний
│   ├── service/      # бизнес-логика
│   ├── storage/      # реализация PostgreSQL
│   └── models/       # структуры данных
├── migrations/       # SQL-миграции
├── config/           # конфигурация через env
└── cf-worker/        # Cloudflare Worker (опциональный прокси)
```

## Запуск

### Требования

- Docker и docker-compose
- Токен Telegram-бота от [@BotFather](https://t.me/BotFather)
- Свой `chat_id` — узнать у [@userinfobot](https://t.me/userinfobot)

### Установка

1. Клонировать репозиторий

   ```bash
   git clone https://github.com/woka00/go-reminder-bot.git
   cd go-reminder-bot
   ```

2. Заполнить переменные окружения

   ```bash
   cp .env.example .env
   ```

   ```env
   BOT_TOKEN=123456:ABC-DEF
   ALLOWED_CHAT_ID=123456789
   NOTIFY_CHAT_ID=123456789
   DATABASE_URL=postgres://reminder:reminder@postgres:5432/reminder?sslmode=disable
   TIMEZONE=Europe/Moscow

   # Опционально, если api.telegram.org недоступен напрямую:
   # TELEGRAM_API_BASE_URL=https://your-worker.workers.dev
   ```

3. Запустить

   ```bash
   make up      # docker compose up -d --build
   make logs    # следить за выводом
   ```

### Полезные команды Makefile

```bash
make build         # локальная сборка бинарника
make run           # запустить локально (с .env)
make up / down     # docker compose поднять / остановить
make restart       # перезапустить только бот
make logs          # follow логов бота
make psql          # открыть psql в контейнере postgres
make fmt / vet     # форматирование и статанализ
```

## Прокси через Cloudflare Worker

Если бот деплоится из сети, где `api.telegram.org` недоступен, в [cf-worker/](cf-worker/)
лежит готовый Worker-прокси. Лимит бесплатного тарифа Cloudflare Workers — 100 000
запросов в день, для long-polling бота это ~3% использования.

Быстрая настройка:

```bash
cd cf-worker
npm i -g wrangler
wrangler login
wrangler secret put BOT_TOKEN      # вставить токен бота
wrangler deploy
```

Полученный URL прописать в `.env` как `TELEGRAM_API_BASE_URL`.
Подробные инструкции — внутри [cf-worker/worker.js](cf-worker/worker.js).
