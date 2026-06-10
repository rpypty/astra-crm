# Backend Core Domain

Рабочая папка:

```text
astra-crm-backend
```

Цель milestone: реализовать основные операции без CSV/reconciliation-heavy логики: тимлид управляет трейдерами и реквизитами, трейдер берет реквизит в работу, ведет turnover entries, создает payout orders и transfers.

Implementation status:

- 2026-06-08: BE-201..BE-206 implemented in `astra-crm-backend`.
- Next epic boundary starts at backend imports/reconciliation; do not continue from this file when asked to stop after current epic.

## BE-201. Teamlead Traders CRUD

Контекст:

```text
docs/domain-model.md
docs/api-outline.md
docs/security-and-audit.md
```

Задача:

- реализовать list/create/get/patch/delete traders;
- создать `User(role=trader)` и `TraderProfile`;
- поддержать salary rate in bps;
- поддержать `external_worker_name`;
- reset password endpoint;
- soft delete/disable вместо hard delete.

Endpoints:

```text
GET    /api/v1/teamlead/traders
POST   /api/v1/teamlead/traders
GET    /api/v1/teamlead/traders/{traderId}
PATCH  /api/v1/teamlead/traders/{traderId}
DELETE /api/v1/teamlead/traders/{traderId}
POST   /api/v1/teamlead/traders/{traderId}/reset-password
```

Tests:

- teamlead can create trader;
- duplicate login rejected;
- password never returned;
- salary bps validates non-negative;
- password reset audited without password/hash in audit payload;
- trader cannot call endpoints.

Done:

- endpoints work;
- audit is written;
- API responses explicit and typed.

## BE-202. Teamlead Requisites and Assignment History

Контекст:

```text
docs/domain-model.md
docs/api-outline.md
docs/security-and-audit.md
```

Задача:

- CRUD requisites;
- assign/unassign/reassign requisite to trader;
- preserve assignment history;
- enforce one active assignment;
- base requisite fields: phone, method, proxy only;
- card number/holder name not stored on base requisite.

Endpoints:

```text
GET    /api/v1/teamlead/requisites
POST   /api/v1/teamlead/requisites
GET    /api/v1/teamlead/requisites/{requisiteId}
PATCH  /api/v1/teamlead/requisites/{requisiteId}
DELETE /api/v1/teamlead/requisites/{requisiteId}
POST   /api/v1/teamlead/requisites/{requisiteId}/assign
POST   /api/v1/teamlead/requisites/{requisiteId}/unassign
GET    /api/v1/teamlead/requisites/{requisiteId}/assignment-history
```

Tests:

- assign creates active assignment;
- reassign closes previous assignment and creates new one;
- two active assignments impossible;
- assignment to disabled trader rejected;
- archive uses soft delete/status;
- all mutations audited.

Done:

- transaction boundary obvious for reassign;
- DB constraint backs service invariant.

## BE-203. Trader Current Shift and Take Requisite

Контекст:

```text
docs/domain-model.md
docs/ux-flows.md
docs/api-outline.md
```

Задача:

- get current shift;
- trader sees assigned requisites;
- take assigned requisite into work;
- create open shift automatically if missing;
- create `shift_requisite` with card number and holder name;
- forbid taking unassigned requisite;
- forbid modifying closed shift.

Endpoints:

```text
GET  /api/v1/trader/shift/current
GET  /api/v1/trader/requisites
POST /api/v1/trader/requisites/{requisiteId}/take
PATCH /api/v1/trader/shift-requisites/{shiftRequisiteId}
GET  /api/v1/trader/shift-requisites
```

Tests:

- first take creates shift;
- second take uses existing open shift;
- trader has only one open shift;
- unassigned requisite rejected;
- card/holder required;
- mutation audited.

Done:

- lifecycle starts exactly as docs define;
- no manual start shift endpoint.

## BE-204. Cumulative Turnovers

Контекст:

```text
docs/domain-model.md
docs/reconciliation-rules.md
docs/api-outline.md
```

Задача:

- create turnover entry for shift requisite;
- entries are append-only;
- expose latest turnover per shift requisite;
- expose turnover history.

Endpoints:

```text
GET  /api/v1/trader/shift/current/turnovers
POST /api/v1/trader/shift/current/turnovers
GET  /api/v1/trader/shift-requisites/{shiftRequisiteId}/turnovers
```

Tests:

- amount must be non-negative;
- latest entry is selected by created_at/id deterministic ordering;
- older turnover ignored by latest calculation;
- trader cannot add turnover to another trader shift requisite;
- mutation audited.

Done:

- service exposes data needed by reconciliation later.

## BE-205. Manual Payout Orders and Transfers

Контекст:

```text
docs/domain-model.md
docs/reconciliation-rules.md
docs/api-outline.md
```

Задача:

- create/update/cancel manual payout order;
- add/delete intermediate transfers if allowed;
- compute paid/remaining;
- mark payout paid when transfer sum equals amount;
- reject transfer over remaining;
- use transaction and row lock when adding transfer.

Endpoints:

```text
GET    /api/v1/trader/payouts
POST   /api/v1/trader/payouts
GET    /api/v1/trader/payouts/{payoutId}
PATCH  /api/v1/trader/payouts/{payoutId}
DELETE /api/v1/trader/payouts/{payoutId}
POST   /api/v1/trader/payouts/{payoutId}/transfers
DELETE /api/v1/trader/payouts/{payoutId}/transfers/{transferId}
```

Tests:

- partial transfer keeps payout in progress;
- exact transfer marks payout paid;
- overpay rejected;
- outbound calculations use transfers, not payout headers;
- mutations audited.

Done:

- payout domain is ready for outbound reconciliation.

## BE-206. Shift Close Checklist Skeleton

Контекст:

```text
docs/reconciliation-rules.md
docs/api-outline.md
docs/ux-flows.md
```

Задача:

- implement checklist endpoint;
- implement close endpoint with current known blockers;
- initially reconciliation blockers can rely on status fields;
- unpaid payout blocker must be real.

Endpoints:

```text
GET  /api/v1/trader/shift/current/checklist
POST /api/v1/trader/shift/current/close
```

Rules:

- cannot close without inbound imported and OK;
- cannot close without outbound imported and OK;
- cannot close with unpaid payout orders;
- accepted mismatch means final status `closed_with_discrepancy`;
- closed shift cannot reopen.

Tests:

- close blocked without imports;
- close blocked with unpaid payouts;
- close succeeds when statuses OK and payouts paid;
- accepted mismatch sets discrepancy status;
- close audited.

Done:

- close logic is ready to be connected to reconciliation service.
