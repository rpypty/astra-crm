# Backend Imports and Reconciliation

Рабочая папка:

```text
astra-crm-backend
```

Цель milestone: реализовать CSV parser, import/reimport semantics, active scope model и сверки, которые делают продукт полезным.

Implementation status:

- 2026-06-08: BE-301 implemented in `astra-crm-backend` as DB-independent CSV parser with golden fixtures.
- 2026-06-08: BE-302 transactional repository/service core and HTTP upload endpoints implemented.
- 2026-06-08: BE-303 trader inbound reconciliation core, import/turnover hooks, latest endpoint and accept endpoint implemented.
- 2026-06-09: BE-304 trader outbound reconciliation implemented with payout-transfer actuals, accept endpoint and close blockers.
- 2026-06-09: BE-305 teamlead period inbound reconciliation implemented with total, worker and order-level mismatch items.
- 2026-06-09: BE-306 orders, dashboards and import history read models implemented for trader/teamlead inbound/outbound endpoints.

## BE-301. CSV Parser

Контекст:

```text
docs/csv-imports.md
docs/reconciliation-rules.md
docs/testing-strategy.md
```

Задача:

- parse CSV with `|` delimiter;
- support teamlead and trader column sets;
- handle optional columns;
- parse `None` as null;
- parse dates `DD.MM.YYYY HH:mm:ss`;
- parse money exactly without float;
- map statuses;
- reject duplicate `innerId` inside one file with row numbers;
- return parsed rows and validation errors.

Testdata:

```text
testdata/csv/teamlead.csv
testdata/csv/trader.csv
```

Tests:

- teamlead columns parse;
- trader columns parse;
- optional fields parse;
- `None` becomes null;
- `3001.0` becomes `300100` minor units;
- unknown status stored as unknown and not counted as success;
- duplicate `innerId` rejects entire import.

Done:

- parser has no DB dependency;
- parser tests are deterministic;
- no float used for money.

## BE-302. Import and Reimport Service

Контекст:

```text
docs/domain-model.md
docs/csv-imports.md
docs/database-design.md
docs/reconciliation-rules.md
```

Задача:

- create import_batch;
- save import_rows with raw_payload_json;
- upsert external_orders by `team_id + direction + external_inner_id`;
- deactivate previous active order_scope_items for same scope;
- mark previous active import batch as superseded;
- create new active order_scope_items;
- preserve historical rows/batches;
- write audit event;
- trigger reconciliation recalculation hook.

Scopes:

```text
trader_shift + shift_id + direction
teamlead_period + accounting_period_id + direction
```

Tests:

- first import creates active scope items;
- reimport old A/B/C -> new A/C/D leaves A/C/D active;
- B remains historical but inactive;
- changed A updates external_order;
- previous batch superseded;
- duplicate innerId fails before applying active set;
- transaction rollback leaves old active set intact on failure.

Done:

- latest active scope wins calculations;
- no hard delete of import history.

## BE-303. Trader Inbound Reconciliation

Контекст:

```text
docs/reconciliation-rules.md
docs/domain-model.md
```

Задача:

- calculate expected amount from active inbound CSV success/corrected orders;
- calculate actual amount from latest cumulative turnover per shift requisite;
- create reconciliation_run;
- update shift inbound reconciliation status;
- expose latest run endpoint;
- accept mismatch with required comment.

Endpoints:

```text
GET  /api/v1/trader/inbound/reconciliation/latest
POST /api/v1/trader/inbound/reconciliation/{runId}/accept
```

Tests:

- matched when expected equals actual;
- mismatch when diff exists;
- corrected counted as success;
- latest turnover only;
- empty comment rejected;
- accepted mismatch updates run and shift status;
- audit written.

Done:

- import/reimport and turnover changes can rerun reconciliation.

## BE-304. Trader Outbound Reconciliation

Контекст:

```text
docs/reconciliation-rules.md
docs/domain-model.md
```

Задача:

- expected amount: active outbound CSV payout orders;
- actual amount: sum manual_payout_transfers for shift;
- create reconciliation_run;
- update shift outbound reconciliation status;
- include unpaid payout blocker item where useful;
- accept mismatch with required comment.

Tests:

- matched when CSV amount equals transfer sum;
- mismatch when diff exists;
- payout headers alone do not count as actual;
- unpaid payout blocks close even if reconciliation amount matches;
- accepted mismatch requires comment.

Done:

- shift close can rely on outbound status and unpaid payout check.

## BE-305. Teamlead Period Inbound Reconciliation

Контекст:

```text
docs/reconciliation-rules.md
docs/domain-model.md
docs/api-outline.md
```

Задача:

- compare teamlead active inbound period CSV vs trader active inbound shift imports;
- compare total success amount/count;
- compare by `workerName`/trader;
- compare by `external_inner_id`;
- create reconciliation_items.

Issue types:

```text
missing_in_trader_import
extra_in_trader_import
amount_mismatch
status_mismatch
worker_mismatch
total_amount_mismatch
```

Endpoints:

```text
GET /api/v1/teamlead/inbound/reconciliation/latest
GET /api/v1/teamlead/periods/{periodId}/reconciliation/inbound
GET /api/v1/teamlead/periods/{periodId}/reconciliation/items
```

Tests:

- matched period;
- missing in trader import;
- extra in trader import;
- amount mismatch;
- status mismatch;
- worker mismatch;
- corrected counted as success.

Done:

- teamlead can see actionable period mismatch details.

## BE-306. Orders, Dashboards, Import History

Контекст:

```text
docs/api-outline.md
docs/reconciliation-rules.md
docs/ui-pages-and-dialogs.md
```

Задача:

- implement trader inbound/outbound dashboards;
- implement teamlead inbound/outbound dashboards;
- implement order lists with filters;
- implement import history where needed;
- return status breakdown and unknown statuses warnings.

Endpoints:

```text
GET /api/v1/trader/inbound/dashboard
GET /api/v1/trader/inbound/orders
GET /api/v1/trader/outbound/dashboard
GET /api/v1/trader/outbound/orders
GET /api/v1/teamlead/inbound/dashboard
GET /api/v1/teamlead/inbound/orders
GET /api/v1/teamlead/outbound/dashboard
GET /api/v1/teamlead/outbound/orders
```

Filters:

```text
dateFrom
dateTo
traderId
workerName
requisite
methodType
status
amountFrom
amountTo
page
pageSize
sort
```

Done:

- list responses are paginated;
- money and counts are exact;
- frontend has enough data for dashboards and tables.
