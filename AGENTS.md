# AGENTS.md — p2p-crm

Ты работаешь в репозитории **p2p-crm**.

Проект: CRM для тимлидов и P2P-трейдеров.

Основной язык общения с владельцем проекта: русский. Комментарии в коде допускаются на английском или русском, но доменные документы и пользовательские тексты лучше вести на русском.

---

## 1. Product summary

p2p-crm — закрытая CRM для операционной работы P2P-команд.

MVP покрывает две роли:

- `TEAMLEAD` — управляет трейдерами, реквизитами, импортирует CSV за период, смотрит ордера/аналитику/аудит.
- `TRADER` — работает со своими реквизитами, фиксирует ежедневные данные по реку, cumulative-обороты, ручные выплаты, импортирует CSV в конце смены и закрывает смену после сверки.

Админскую/superadmin-панель в MVP не делаем.

Главный доменный агрегат: **TraderShift**.

---

## 2. Tech stack

Backend:

- Go modular monolith.
- `chi` router.
- `pgx` for PostgreSQL.
- `sqlc` for typed SQL code generation.
- `goose` migrations.
- `slog` for structured logging.
- REST API + OpenAPI.
- No GORM unless explicitly requested.

Database:

- PostgreSQL.
- Money as integer/bigint, not float.
- Percent rates as basis points (`salary_rate_bps`).
- Raw CSV payloads as JSONB.

Frontend:

- React + TypeScript + Vite.
- TanStack Router.
- TanStack Query.
- TanStack Table.
- React Hook Form + Zod.
- shadcn/ui + Tailwind CSS.
- Recharts for dashboards.

Auth:

- Login/password.
- Server-side sessions.
- Session token in `httpOnly`, `secure`, `sameSite` cookie.
- Store only session token hash in DB.
- RBAC middleware.

Deploy MVP:

- Docker Compose.
- VPS.
- PostgreSQL.
- Backend container.
- Frontend static SPA.
- Caddy or Nginx.

---

## 3. Go coding preferences

- Prefer `int64` over `int` for IDs, counters, money, and domain numbers.
- Keep domain logic explicit and testable.
- Avoid hidden ORM magic.
- Keep transaction boundaries obvious.
- Repository layer should expose meaningful domain operations, not random DB helpers.
- Use context-aware methods: `ctx context.Context`.
- Avoid floats for money.
- Avoid storing secrets/passwords in logs or audit.
- Prefer clear enum-like string constants for statuses.

Suggested backend layout:

```text
/backend
  /cmd/api
  /internal
    /auth
    /users
    /teams
    /requisites
    /shifts
    /imports
    /orders
    /payouts
    /reconciliation
    /audit
    /dashboard
  /migrations
  /sqlc
```

---

## 4. Domain rules — non-negotiable

### Requisites

- A teamlead creates requisites.
- Base requisite fields: phone, method, proxy, assigned trader.
- Card number / holder name are NOT base requisite fields. They belong to a trader shift.
- Requisite assignment history must be preserved.
- A requisite can have only one active assignment at a time.
- Requisites should be soft-deleted/archived, not hard-deleted, once used.

### Trader shift

- A trader does not manually press “start shift”.
- Shift starts automatically when trader takes the first assigned requisite into work by filling daily payment details.
- Shift ends after CSV import and successful/accepted reconciliation.
- Closed shift cannot be reopened.
- Shift can be `closed_with_discrepancy` only if mismatch was accepted with a required comment.

### Turnovers

- Requisite turnovers are cumulative.
- Trader enters current accumulated turnover for a requisite.
- For reconciliation, use the latest turnover entry per `shift_requisite`.

### Inbound success statuses

Successful inbound turnover includes:

- `hand_success`
- `corrected`

`corrected` must be saved and counted as success. Store `oldAmount` but use current `amount` for calculations unless requirements change.

### CSV imports

- CSV separator: `|`.
- Main external unique key: `innerId`.
- Teamlead and trader may upload CSV multiple times.
- Reimport fully replaces the active order set for the corresponding scope:
  - trader shift + direction;
  - teamlead period + direction.
- Preserve historical import batches and raw rows.
- Do not hard-delete old imports for replacement.
- The latest active import scope wins for calculations.
- Duplicated `innerId` inside a single CSV should be rejected as import error unless product owner changes this rule.
- Reimport must trigger recalculation of external orders, active scope items, reconciliation and shift/period statuses.

### Payouts

- Trader creates manual payout orders.
- Manual payout order can be filled by multiple intermediate transfers.
- Outbound reconciliation compares CSV payout orders with intermediate manual transfers, not just payout order headers.
- A payout order cannot be considered paid until sum of transfers equals its amount.
- Shift cannot be closed if there are not-fully-paid manual payout orders.

### Reconciliation

- Any mismatch requires a comment.
- Trader can accept mismatch with comment.
- If any shift reconciliation is accepted with comment, final shift status is `closed_with_discrepancy`.
- Teamlead period inbound reconciliation should compare teamlead CSV vs trader imports at total, worker, and order levels.

### Audit

Audit all mutations:

- create/edit/delete trader;
- password reset;
- salary percent change;
- create/edit/delete requisite;
- assign/reassign requisite;
- take requisite into work;
- change card/holder details;
- add turnover;
- create/edit/delete manual payout;
- add/delete payout transfer;
- import/reimport CSV;
- accept mismatch;
- close shift;
- close accounting period.

Never log plaintext passwords, session tokens, or password hashes.

---

## 5. Core entities

See `docs/domain-model.md` for details. Core entities:

```text
Team
User
TraderProfile
Requisite
RequisiteAssignment
TraderShift
ShiftRequisite
RequisiteTurnoverEntry
ImportBatch
ImportRow
ExternalOrder
OrderScopeItem
ManualPayoutOrder
ManualPayoutTransfer
AccountingPeriod
ReconciliationRun
ReconciliationItem
AuditLog
AuthSession
```

---

## 6. Implementation style for Codex

When implementing tasks:

1. Read relevant docs first.
2. State a short implementation plan.
3. Make small, coherent changes.
4. Add or update tests for domain rules.
5. Do not silently change product behavior.
6. Do not invent fields that conflict with domain docs.
7. If there is ambiguity, choose the safest MVP-compatible option and document it in `docs/open-questions.md` or task notes.
8. Keep database constraints aligned with domain invariants.
9. Keep API responses explicit and typed.
10. Prefer backend validation even if frontend validation already exists.

---

## 7. Definition of done

For backend tasks:

- Migrations are created if schema changes.
- SQL queries are added/updated.
- Domain service logic is covered by tests.
- Reconciliation/import edge cases are tested.
- Errors are explicit and user-safe.
- Audit is written for mutations.
- Code compiles.

For frontend tasks:

- Empty/loading/error states are implemented.
- Forms have validation.
- Tables support filters/search where required.
- Destructive actions require confirmation.
- Mismatch states are visually obvious.
- UX follows docs/ui-pages-and-dialogs.md and docs/frontend-design-system.md.

---

## 8. Important docs

Before starting serious work, read:

- `docs/product-context.md`
- `docs/domain-model.md`
- `docs/csv-imports.md`
- `docs/reconciliation-rules.md`
- `docs/database-design.md`
- `docs/ux-flows.md`
- `docs/architecture-decisions.md`
