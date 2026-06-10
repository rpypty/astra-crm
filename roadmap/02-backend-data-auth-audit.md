# Backend Data, Auth, RBAC, Audit

Рабочая папка:

```text
astra-crm-backend
```

Цель milestone: создать базовую схему PostgreSQL, session auth, RBAC, team scoping и audit writer до активной доменной разработки.

## BE-101. MVP Schema Migrations

Контекст:

```text
AGENTS.md
docs/domain-model.md
docs/database-design.md
docs/reconciliation-rules.md
docs/security-and-audit.md
```

Задача:

- создать goose migration up/down для MVP schema;
- использовать `BIGSERIAL` или UUID последовательно, для MVP предпочтительно `BIGSERIAL`;
- деньги хранить в `amount_minor BIGINT`;
- статусы хранить `TEXT` + `CHECK`;
- добавить partial unique indexes.

Таблицы:

```text
teams
users
trader_profiles
auth_sessions
requisites
requisite_assignments
trader_shifts
shift_requisites
requisite_turnover_entries
manual_payout_orders
manual_payout_transfers
accounting_periods
import_batches
import_rows
external_orders
order_scope_items
reconciliation_runs
reconciliation_items
audit_logs
```

Особые constraints:

- one active assignment per requisite;
- one open/closing shift per trader;
- `scope_type` consistency for import/order scope tables;
- `date_to >= date_from`;
- non-negative money where appropriate.

Done:

- migration has up/down;
- schema matches `docs/database-design.md` or deviation documented;
- migration applies on empty DB;
- migration rolls back.

## BE-102. sqlc Baseline Queries

Контекст:

```text
docs/database-design.md
docs/api-outline.md
```

Задача:

- добавить sqlc queries для foundation tables;
- сгенерировать typed DB layer;
- не превращать repository слой в random DB helpers.

Минимальный набор:

- users by login/id;
- auth sessions create/find/revoke;
- teams by id;
- audit insert/list;
- basic requisites/traders lookup for auth/domain tasks.

Done:

- generated code компилируется;
- repository методы принимают `ctx context.Context`;
- `go test ./...` проходит.

## BE-103. Password Hashing and Sessions

Контекст:

```text
docs/security-and-audit.md
docs/api-outline.md
```

Задача:

- реализовать password hashing через bcrypt или argon2id;
- реализовать login/logout/me;
- создать session token, в DB хранить только hash;
- cookie: `httpOnly`, `secure` configurable, `sameSite`;
- logout revokes session.

Endpoints:

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/me
```

Done:

- password/hash/token не логируются;
- raw token не хранится в DB;
- disabled/deleted users не логинятся;
- tests cover login/logout/me.

## BE-104. RBAC and Team Scope Middleware

Контекст:

```text
docs/security-and-audit.md
docs/api-outline.md
```

Задача:

- добавить auth middleware;
- добавить role middleware;
- добавить current user/team context;
- team_id брать из session user, не из request body.

Правила:

- `TEAMLEAD` может работать с teamlead endpoints своей команды;
- `TRADER` может работать только со своими trader endpoints;
- unauthenticated request rejected.

Done:

- trader cannot access `/teamlead/*`;
- unauthenticated request rejected;
- user from another team cannot access data;
- tests cover role boundaries.

## BE-105. Audit Writer and Redaction

Контекст:

```text
docs/security-and-audit.md
docs/domain-model.md
```

Задача:

- добавить audit service;
- добавить append-only insert;
- добавить redaction helper;
- договориться о masking card number in audit payload.

Минимальные actions:

```text
user.created
user.updated
user.password_reset
requisite.created
requisite.updated
requisite.assigned
shift.created
shift.requisite_taken
shift.turnover_added
manual_payout.created
manual_payout.transfer_added
import.applied
reconciliation.accepted_with_comment
shift.closed
```

Done:

- audit writer используется в следующих mutation tasks;
- sensitive keys are redacted;
- tests assert no password/password_hash/token fields in audit JSON.

