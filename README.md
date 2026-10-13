# go-help-cs

**Автоматическая обработка тикетов поддержки с помощью локальной LLM: Go-сервис забирает тикеты из Freshdesk, готовит ответы через Ollama и поднимает тревогу в Telegram по критичным системным сообщениям.**

*Go service that triages Freshdesk support tickets with a local LLM (Ollama): drafts replies for customer tickets, analyzes system alerts and escalates critical ones to Telegram. RabbitMQ queue, PostgreSQL, scheduler.*

## Как работает

```
 ticker (каждые N минут)                  RabbitMQ                 worker
 scheduler ──► Freshdesk API: новые ──► ticket_queue ──► сохранить тикет в PostgreSQL
              тикеты, тип по email                        │
              отправителя (клиент / система)              ├─ клиентский: LLM (CS_Bro) готовит ответ
                                                          │              ──► публичный комментарий в Freshdesk
                                                          └─ системный:  LLM (Support_bro) + правила
                                                                         ──► приватный комментарий
                                                                         ──► алерт в Telegram, если критично
```

- **Планировщик** (`time.Ticker`) раз в N минут забирает новые тикеты из Freshdesk и делит их на клиентские и системные по списку email-адресов.
- **Очередь RabbitMQ** развязывает получение и обработку: тикеты не теряются, если LLM отвечает медленно.
- **Две роли LLM** на базе llama3.1 с собственными Modelfile ([models/](models/)): `CS_Bro` отвечает клиентам, `Support_bro` разбирает системные проблемы и предлагает технические решения.
- **Анализатор системных сообщений:** выжимка через LLM плюс правила для критичных признаков (ошибки, сбои, безопасность, клиенты без заказов N дней).
- **Эскалация в Telegram:** критичные системные тикеты уходят дежурному с информацией о клиенте.
- **PostgreSQL** хранит тикеты, сгенерированные ответы и статус обработки.
- Graceful shutdown по SIGINT и SIGTERM.

## Стек

Go · RabbitMQ (streadway/amqp) · PostgreSQL (lib/pq) · Ollama (llama3.1) · Freshdesk API · Telegram Bot API · Docker Compose

## Запуск

```bash
cp env.example .env      # Freshdesk, Telegram, список системных email
ollama create CS_Bro -f models/CS_Bro.modelfile
ollama create Support_bro -f models/Support_bro.modelfile
docker compose up --build
```

Ollama запускается на хосте, сервис обращается к нему по `OLLAMA_BASE_URL` (по умолчанию `http://host.docker.internal:11434`).

## Структура

```
cmd/main.go            точка входа, graceful shutdown
internal/scheduler     периодический опрос Freshdesk
internal/queue         RabbitMQ: публикация и потребление
internal/worker        обработка тикета
internal/analyzer      анализ системных сообщений
internal/ollama        клиент Ollama и промпты
internal/freshdesk     клиент Freshdesk API
internal/telegram      алерты
internal/database      PostgreSQL
models/                Modelfile для двух ролей LLM
```
