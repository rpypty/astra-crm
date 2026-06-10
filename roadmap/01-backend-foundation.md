# Backend Foundation

Рабочая папка:

```text
astra-crm-backend
```

Цель milestone: получить минимальный backend, который компилируется, запускается, имеет health endpoint, конфиг, logger, graceful shutdown и подготовлен к PostgreSQL, goose и sqlc.

## BE-001. Initialize Backend Repository

Контекст:

```text
AGENTS.md
docs/architecture-decisions.md
docs/api-outline.md
docs/testing-strategy.md
```

Задача:

- инициализировать Go module;
- создать базовую структуру modular monolith;
- добавить README с dev commands;
- добавить `.gitignore`;
- подготовить пустые доменные папки под MVP modules.

Рекомендуемая структура:

```text
cmd/api
internal/config
internal/httpserver
internal/platform/logger
internal/platform/postgres
internal/auth
internal/users
internal/teams
internal/requisites
internal/shifts
internal/imports
internal/orders
internal/payouts
internal/reconciliation
internal/audit
internal/dashboard
migrations
sqlc
```

Implementation notes:

- Go version выбрать текущую стабильную, зафиксировать в `go.mod`.
- Не добавлять GORM.
- ID/domain numbers в публичных моделях планировать как `int64`.
- Пока не реализовывать бизнес-логику, только scaffold.

Done:

- `go test ./...` проходит;
- README объясняет local run;
- структура модулей соответствует docs.

## BE-002. HTTP Server, Config, Logger

Контекст:

```text
docs/architecture-decisions.md
docs/security-and-audit.md
```

Задача:

- добавить config loading из environment;
- добавить structured logging через `slog`;
- добавить `chi` router;
- добавить graceful shutdown;
- добавить request logging middleware без cookies/body/passwords.

Минимальные config поля:

```text
APP_ENV
HTTP_ADDR
DATABASE_URL
SESSION_COOKIE_NAME
SESSION_SECURE
```

Done:

- backend запускается локально;
- есть request id;
- request logs не содержат cookie/body;
- `go test ./...` проходит.

## BE-003. Health Endpoints

Контекст:

```text
docs/api-outline.md
```

Задача:

- добавить `GET /health`;
- добавить `GET /ready`, если DB connection уже есть;
- вернуть explicit JSON.

Пример:

```json
{
  "status": "ok"
}
```

Done:

- `GET /health` работает без DB;
- endpoint покрыт test;
- ошибки readiness безопасны.

## BE-004. PostgreSQL Connection Skeleton

Контекст:

```text
docs/architecture-decisions.md
docs/database-design.md
```

Задача:

- добавить pgx pool initialization;
- добавить shutdown pool close;
- добавить DB ping для readiness;
- не создавать schema руками в коде.

Done:

- приложение может стартовать с DB;
- приложение может стартовать без DB только если режим явно dev/test или health не требует DB;
- `go test ./...` проходит.

## BE-005. Goose and sqlc Preparation

Контекст:

```text
docs/database-design.md
docs/testing-strategy.md
```

Задача:

- добавить `goose` migrations folder;
- добавить `sqlc.yaml`;
- добавить базовую структуру SQL query folders;
- описать commands в README.

Done:

- есть место для migrations;
- есть место для sqlc queries;
- будущая задача по schema migrations не требует перестройки layout.

## BE-006. Common API Error Model

Контекст:

```text
docs/api-outline.md
docs/security-and-audit.md
```

Задача:

- добавить common error response model;
- добавить helpers для validation/domain/unauthorized/forbidden/not_found errors;
- добавить middleware для panic recovery с безопасным ответом.

Done:

- API errors имеют единый формат;
- technical details не уходят клиенту;
- request_id есть в logs.

