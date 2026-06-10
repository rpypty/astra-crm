# Codex Task Prompts

Use these prompts after placing `AGENTS.md` and `docs/` into the repo.

---

## 1. Bootstrap backend

```text
Прочитай AGENTS.md и docs/*.md.

Задача: создай backend scaffold для p2p-crm на Go.

Goal:
- modular monolith structure;
- chi router;
- config loading;
- pgx connection;
- health endpoint;
- slog logger;
- graceful shutdown.

Constraints:
- no GORM;
- use int64 for IDs/domain numbers;
- code should be ready for sqlc/goose later;
- keep modules aligned with docs/domain-model.md.

Done when:
- backend compiles;
- GET /health returns ok;
- README has run commands.
```

---

## 2. Create database migrations

```text
Прочитай AGENTS.md, docs/domain-model.md, docs/database-design.md, docs/reconciliation-rules.md.

Задача: создай goose migrations для MVP схемы p2p-crm.

Особое внимание:
- users/trader_profiles;
- requisites/requisite_assignments;
- trader_shifts/shift_requisites/turnover_entries;
- manual_payout_orders/manual_payout_transfers;
- import_batches/import_rows/external_orders/order_scope_items;
- accounting_periods;
- reconciliation_runs/reconciliation_items;
- audit_logs/auth_sessions;
- partial unique indexes: one open shift per trader, one active assignment per requisite;
- money as bigint minor units;
- no floats for money.

Done when:
- migrations have up/down;
- constraints and indexes are present;
- schema matches docs unless you document a justified deviation.
```

---

## 3. Implement CSV parser

```text
Прочитай AGENTS.md и docs/csv-imports.md.

Задача: реализуй CSV parser для teamlead/trader CSV.

Goal:
- delimiter |;
- optional column handling;
- parse None as null;
- parse dates DD.MM.YYYY HH:mm:ss;
- parse money without float;
- normalize statuses;
- reject duplicate innerId inside same file;
- return parsed rows and validation errors.

Add tests using sanitized testdata based on observed columns.

Done when:
- parser tests pass;
- teamlead/trader column sets are supported;
- duplicate innerId returns clear error with row numbers.
```

---

## 4. Implement import/reimport service

```text
Прочитай AGENTS.md, docs/domain-model.md, docs/csv-imports.md, docs/reconciliation-rules.md.

Задача: реализуй ImportService для CSV import/reimport.

Goal:
- create import_batch;
- save import_rows;
- upsert external_orders by team_id + direction + external_inner_id;
- deactivate previous active order_scope_items for the same scope;
- mark previous active batch superseded;
- create new active order_scope_items;
- trigger reconciliation recalculation;
- write audit event.

Constraints:
- all in DB transaction;
- historical rows preserved;
- latest active scope set wins;
- no hard delete of import history.

Done when:
- reimport tests pass: old A/B/C -> new A/C/D results in active A/C/D and inactive B;
- audit is written;
- reconciliation is recalculated.
```

---

## 5. Implement trader shift lifecycle

```text
Прочитай AGENTS.md, docs/domain-model.md, docs/ux-flows.md.

Задача: реализуй backend use cases for trader shift lifecycle.

Use cases:
- get current shift;
- take assigned requisite into work;
- create shift automatically if missing;
- add cumulative turnover entry;
- get latest turnover per shift requisite;
- close shift with checklist validation.

Constraints:
- trader can have only one open shift;
- trader can take only assigned requisite;
- closed shift cannot reopen;
- close requires inbound/outbound reconciliation matched or accepted with comment;
- close requires all payout orders fully paid;
- audit all mutations.

Done when:
- unit/integration tests cover lifecycle and blockers.
```

---

## 6. Implement payout domain

```text
Прочитай AGENTS.md, docs/domain-model.md, docs/reconciliation-rules.md.

Задача: реализуй manual payout domain.

Use cases:
- create manual payout order;
- update/cancel payout order;
- add intermediate transfer;
- delete transfer if allowed;
- compute paid/remaining;
- mark payout paid when remaining is zero.

Constraints:
- transfer sum cannot exceed payout amount;
- outbound reconciliation uses transfer sum;
- shift cannot close if payout incomplete;
- use transactions and row locks where needed;
- audit all mutations.

Done when:
- tests cover partial/complete/overpay cases.
```

---

## 7. Implement reconciliation service

```text
Прочитай AGENTS.md и docs/reconciliation-rules.md.

Задача: реализуй ReconciliationService.

Functions:
- reconcile trader shift inbound;
- reconcile trader shift outbound;
- reconcile teamlead period inbound;
- accept mismatch with required comment.

Rules:
- inbound trader: expected = CSV success/corrected amount; actual = latest cumulative turnovers;
- outbound trader: expected = CSV payout amount; actual = sum manual payout transfers;
- teamlead period inbound: compare teamlead CSV vs trader active imports by total, trader, innerId;
- any mismatch comment required to accept;
- create reconciliation_runs and reconciliation_items.

Done when:
- tests cover matched, mismatch, accepted_with_comment, missing/extra/amount/status/worker mismatch.
```

---

## 8. Implement auth/RBAC

```text
Прочитай AGENTS.md и docs/security-and-audit.md.

Задача: реализуй auth module.

Goal:
- login/password;
- password hashing;
- auth_sessions table;
- httpOnly cookie;
- logout/revoke;
- me endpoint;
- role middleware;
- team_id scoping.

Constraints:
- store token hash only;
- never log password/hash/token;
- backend RBAC required.

Done when:
- unauthenticated requests rejected;
- trader cannot access teamlead endpoints;
- sessions can be revoked.
```

---

## 9. Bootstrap frontend

```text
Прочитай AGENTS.md, docs/frontend-design-system.md, docs/ui-pages-and-dialogs.md, docs/ux-flows.md.

Задача: создай frontend scaffold.

Stack:
- React + TypeScript + Vite;
- Tailwind;
- shadcn/ui;
- TanStack Router;
- TanStack Query;
- TanStack Table;
- React Hook Form + Zod.

Goal:
- app shell with role-based sidebar;
- login page;
- placeholder pages for teamlead/trader;
- shared components: PageHeader, StatusBadge, MoneyCell, EmptyState, ConfirmDialog.

Done when:
- frontend runs;
- navigation works;
- design style follows docs.
```

---

## 10. Build teamlead requisites UI

```text
Прочитай docs/ui-pages-and-dialogs.md и docs/frontend-design-system.md.

Задача: реализуй Teamlead Requisites UI.

Features:
- list with search/filter;
- create/edit drawer;
- assign trader;
- assignment history dialog;
- archive/delete confirmation;
- audit details link/placeholders.

UX:
- compact modern table;
- status badges;
- loading/empty/error states;
- validation.

Done when:
- page matches MVP flow;
- no destructive action without confirmation.
```

---

## 11. Build trader shift UI

```text
Прочитай docs/ux-flows.md, docs/ui-pages-and-dialogs.md, docs/frontend-design-system.md.

Задача: реализуй Trader shift UI.

Features:
- my requisites page;
- take requisite dialog;
- current shift banner;
- add cumulative turnover dialog;
- turnover history;
- close shift checklist modal.

Rules:
- explain cumulative amount in UI;
- show blockers for closing shift;
- mismatch states must be obvious.
```

---

## 12. Build import/mismatch UI

```text
Прочитай docs/csv-imports.md, docs/reconciliation-rules.md, docs/ui-pages-and-dialogs.md.

Задача: реализуй reusable CSV import components.

Components:
- ImportCsvDialog;
- ImportResultDialog;
- MismatchAlert;
- AcceptMismatchDialog.

Requirements:
- show reimport warning;
- show rows/counts/totals;
- show expected/actual/diff;
- require comment for mismatch acceptance.
```
