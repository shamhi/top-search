# top-search

Highload-сервис для виджета «Сейчас ищут» на главной странице маркетплейса. Принимает поисковые события из NATS
JetStream, агрегирует поминутные счётчики в Redis ZSET, пересчитывает Top-N каждую секунду и отдаёт кешированный
результат через gRPC.

## 1. Локальный запуск и примеры запросов

### Быстрый старт

```bash
# Копируем example-конфиги (пропускается, если уже есть config.local.yaml и .env)
cp config/config.example.yaml config/config.local.yaml
cp config/.env.example config/.env

make dev         # установка инструментов + генерация proto + docker compose up -d
# или
task dev
```

Стек: сервер (Go) + NATS JetStream + Redis.

| Сервис          | Порт  | Назначение                                       |
|-----------------|-------|--------------------------------------------------|
| top-search      | 50051 | gRPC API                                         |
| top-search      | 2112  | Prometheus `/metrics` + `/healthz`               |
| NATS            | 4222  | JetStream (клиентский порт)                      |
| NATS monitoring | 8222  | HTTP-мониторинг (varz, jsz, connz)               |
| Redis           | 6379  | Ранжирование, кеш топа, stop-слова, дедупликация |

### Конфигурация

Приложение использует `spf13/viper`. Порядок разрешения значений (позднее переопределяет раннее):

```
1. defaults в коде (config/config.go: setDefaults())
2. config/config.yaml
3. config/config.local.yaml   (gitignored, копия config.example.yaml)
4. env vars (OS environment)  → viper.AutomaticEnv()   (ключи с "." и "-" заменяются на "_" → grpc.port = GRPC_PORT)
```

Внутри Docker-контейнера env vars поступают из `env_file: config/.env` (docker-compose). Локально (без Docker) можно
установить их вручную или положиться на YAML-файлы.

Сервис работоспособен без конфигурационных файлов — все параметры имеют значения по умолчанию в `setDefaults()`.

### Продьюсер тестовых данных

```bash
# 1000 событий/сек, 4 воркера, 30 секунд
make produce

# Высокая нагрузка: 10K rps, 8 воркеров
make produce-load

# Тонкая настройка
go run ./cmd/producer -rate 5000 -workers 8 -duration 60s -batch 10
```

Флаги продьюсера:

| Флаг        | По умолчанию          | Описание                          |
|-------------|-----------------------|-----------------------------------|
| `-rate`     | 1000                  | Событий/сек (0 = без ограничений) |
| `-duration` | 30s                   | Длительность (0 = бесконечно)     |
| `-workers`  | 4                     | Горутин-паблишеров                |
| `-batch`    | 5                     | Событий за один publish           |
| `-nats`     | nats://localhost:4222 | NATS URL                          |
| `-subject`  | search.query.created  | Топик JetStream                   |
| `-silent`   | false                 | Скрыть посекундную статистику     |

Продьюсер генерирует реалистичные запросы (русские и английские товары, категории, бренды) через `gofakeit/v7` с
намеренно испорченными форматами (UPPERCASE, лишние пробелы, суффиксы `!!!`, `цена`) для проверки нормализации. 5%
событий — от ботов с повторяющимся `user_id` для проверки дедупликации.

### gRPC вызовы

```bash
# Топ-5 поисковых запросов
grpcurl -plaintext -d '{"limit": 5}' localhost:50051 trend.v1.TrendService/GetTop

# Стриминг топа (обновление каждую секунду)
grpcurl -plaintext -d '{"limit": 5, "update_interval_seconds": 1}' \
  localhost:50051 trend.v1.TrendService/StreamTop

# Управление стоп-листом
grpcurl -plaintext -d '{"word": "спам"}' \
  localhost:50051 trend.v1.TrendService/AddStopWord

grpcurl -plaintext -d '{}' \
  localhost:50051 trend.v1.TrendService/ListStopWords

grpcurl -plaintext -d '{"word": "спам"}' \
  localhost:50051 trend.v1.TrendService/RemoveStopWord
```

gRPC reflection включён для dev-режима — `grpcurl` работает без указания proto-файлов.

### Мониторинг

```bash
# Prometheus метрики
curl http://localhost:2112/metrics

# NATS статус
curl http://localhost:8222/varz | jq

# JetStream статистика (стримы, консьюмеры, pending messages)
curl http://localhost:8222/jsz | jq

# Healthcheck
curl http://localhost:2112/healthz
```

## 2. Контракт данных

NATS-сообщения передаются в формате Protobuf `SearchEvent` из `api/trend/v1/event.proto`.

События публикуются в JetStream-стрим `SEARCH_QUERIES`, subject `search.query.created`.

### Protobuf-схема

```protobuf
message SearchEvent {
  string event_id = 1;
  string query = 2;
  google.protobuf.Timestamp created_at = 3;
  string user_id = 4;
  string session_id = 5;
  string device_id = 6;
  string locale = 7;
  string platform = 8;
}
```

### Пример payload (JSON-представление)

```json
{
  "eventId": "evt-0-0-1748099200123456789",
  "query": "IPHONE 15 Pro Max!!!",
  "createdAt": "2026-05-26T12:00:00Z",
  "userId": "usr-0-42",
  "sessionId": "mobile-app",
  "deviceId": "iphone-15",
  "locale": "ru-RU",
  "platform": "ios"
}
```

### Обоснование полей

| Поле         | Зачем нужно                                                                                          |
|--------------|------------------------------------------------------------------------------------------------------|
| `event_id`   | Уникальный идентификатор события для трассировки и будущей гарантии exactly-once.                    |
| `query`      | Поисковый запрос — основная единица ранжирования. Нормализуется: lowercase, удаление спецсимволов.   |
| `created_at` | Время события на стороне отправителя. Используется для бакетирования, а не время получения брокером. |
| `user_id`    | Ключ дедупликации. Один пользователь → один запрос каждые 30 секунд → блокировка накруток.           |
| `session_id` | Резервный ключ дедупликации (если `user_id` пуст) + метаданные для будущей аналитики.                |
| `device_id`  | Третий резервный ключ дедупликации + метаданные для детекта парсеров/ферм устройств.                 |
| `locale`     | Локаль пользователя для будущих сегментированных топов (ru-RU, en-US и т.д.).                        |
| `platform`   | Платформа (ios, android, web) для будущей сегментации по каналам.                                    |

Иерархия дедупликации: `user_id` → `session_id` → `device_id`. Если все три пусты — событие проходит без проверки (
анонимный трафик).

## 3. Обоснование архитектуры

### Почему Redis ZSET

- **Atomic increments:** `ZINCRBY` — атомарный инкремент счёта запроса в бакете
- **Native ranking:** `ZREVRANGE ... WITHSCORES` — получение Top-N без сортировки на стороне приложения
- **Union:** `ZUNIONSTORE` — мерж пяти поминутных бакетов в один `trend:top` за O(N)
- **TTL:** `EXPIRE` на каждом ключе — автоматическая очистка без фоновых джобов
- **Shared state:** один Redis на несколько инстансов сервиса — бакеты общие, кеш топа локальный

### Почему in-memory cache

`cachedTop` хранится в оперативной памяти и обновляется агрегатором каждую секунду. gRPC `GetTop` читает из кеша без
сетевого вызова к Redis.

- Latency `GetTop`: < 5ms (подтверждено метриками)
- Redis не участвует в горячем read path — выдерживает 10–50x превышение read над write нагрузкой

### Почему поминутные бакеты (1 минута)

- Окно 5 минут = 5 ключей вида `trend:bucket:{yyyyMMddHHmm}`
- Низкое количество ключей → `ZUNIONSTORE` работает быстро
- Точность окна — 1 минута. Для поисковых трендов этого достаточно: никто не ждёт покадровой точности

### Почему TTL на ключах, а не scheduled cleanup

- Бакеты: 10 минут (окно 5 мин + запас)
- `trend:top`: 2 минуты (перезаписывается каждую секунду)
- Dedup: 30 секунд (подавление повторных запросов)
- Нулевые накладные расходы — Redis сам удаляет ключи по expire

### Stop-слова

Хранятся в Redis SET `trend:stopwords` и локальном `map[string]struct{}`. Добавление/удаление пишет в Redis + обновляет
in-memory кеш мгновенно. При старте сервис загружает stop-слова из Redis. Stop-слова применяются на этапе Ingest (
блокировка записи) и на этапе GetTop (фильтрация выдачи).

### Дедупликация (anti-spam)

Ключ: `trend:dedup:{identity}:{query}`. `INCR` + `EXPIRE 30`. Если счётчик > 1 → событие пропускается. Это решает
проблему аномальных всплесков от парсеров/конкурентов: один пользователь с одним запросом учитывается раз в 30 секунд.

## 4. Trade-offs и продуктовые неоднозначности

### Trade-offs

| Решение                          | Выигрыш                                                                   | Плата                                                                  |
|----------------------------------|---------------------------------------------------------------------------|------------------------------------------------------------------------|
| Redis ZSET вместо in-memory      | Атомарные инкременты, native ranking, шардимое состояние между инстансами | Redis — внешняя зависимость для write path                             |
| Кеш топа в памяти                | Read path без сети (< 5ms), выдерживает 10-50x превышение read над write  | Топ может отставать на 1 секунду (интервал агрегации)                  |
| Минутные бакеты                  | Простое окно, низкий key count, быстрый ZUNIONSTORE                       | Точность окна — 1 минута (не по секундам)                              |
| Dedup TTL 30 секунд              | Дешёвое подавление повторных запросов и ботов                             | Не защищает от распределённых атак с разными user_id                   |
| `ZUNIONSTORE` каждую секунду     | Пересчёт окна одним вызовом Redis                                         | При росте уникальных запросов до миллионов — union станет узким местом |
| Локальный `cachedTop` на инстанс | Мгновенное чтение без round-trip                                          | Каждый инстанс держит свою копию; нет глобальной консистентности       |
| Stop-слова in-memory + Redis     | Мгновенное применение, персистентность                                    | При рассинхроне (сбой Redis) применяется локальная копия               |

### Продуктовые неоднозначности

| Неоднозначность                             | Решение                                                                                                    |
|---------------------------------------------|------------------------------------------------------------------------------------------------------------|
| Формат payload не задан                     | Protobuf `SearchEvent` с event_id, query, created_at, user_id, session_id, device_id, locale, platform     |
| Очистка данных не определена                | TTL на всех ключах: бакеты 10 мин, dedup 30 сек, `trend:top` 2 мин. Нулевые накладные расходы              |
| Read-трафик в 10-50 раз выше write          | GetTop читает `cachedTop` из памяти. Redis не участвует в read path вообще                                 |
| «Аномальные накрутки» без конкретики        | Dedup: один identity → один query в 30 сек. Нормализация: спецсимволы удаляются, lowercase                 |
| Сервис стартует пустым                      | Пустой Redis/кеш → `GetTop` возвращает пустой список. Топ появляется по мере поступления событий           |
| Multi-instance поведение не специфицировано | Redis ZSET — shared state. Каждый инстанс агрегирует сам и держит свой `cachedTop`                         |
| Виджет на главной → высокая посещаемость    | gRPC reflection отключён в проде? Включён всегда. Для прода нужно `env: production` + выключить reflection |
| Нет требования к latency                    | Цель: < 10ms на GetTop. Достигнуто: < 5ms (P100). Подтверждено метриками                                   |

## Observability

### Метрики приложения

| Метрика                           | Тип       | Описание                                             |
|-----------------------------------|-----------|------------------------------------------------------|
| `grpc_requests_total`             | Counter   | Всего gRPC запросов (method, code)                   |
| `grpc_request_duration_seconds`   | Histogram | Latency gRPC (buckets: 1–10ms с высоким разрешением) |
| `trend_events_ingested_total`     | Counter   | Принятых событий                                     |
| `trend_dedup_hits_total`          | Counter   | Заблокировано дедупликацией                          |
| `trend_stopword_blocks_total`     | Counter   | Заблокировано стоп-словами                           |
| `trend_empty_query_dropped_total` | Counter   | Отброшено пустых запросов после нормализации         |
| `trend_cache_misses_total`        | Counter   | GetTop cache miss (холодный старт)                   |
| `trend_cache_hits_total`          | Counter   | GetTop cache hit                                     |
| `trend_stopwords_active`          | Gauge     | Текущее количество стоп-слов                         |
| `trend_aggregator_ticks_total`    | Counter   | Циклов агрегации                                     |
| `trend_cached_top_entries`        | Gauge     | Количество записей в кеше топа                       |

## Разработка

```bash
task proto       # генерация protobuf Go-кода
task build       # go build ./...
task test        # go test -race -shuffle=on -count=1 ./...
task vet         # go vet ./...
task lint        # golangci-lint
task dc-upd      # docker compose up -d
task dc-down     # docker compose down
task produce     # продьюсер тестовых данных
task produce-load # стресс-тест 10K rps
```

## Бенчмарки

```bash
go test -bench=. -benchmem ./internal/core/service/
```
