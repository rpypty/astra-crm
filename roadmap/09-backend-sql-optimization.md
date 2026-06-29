# Backend SQL Optimization Stream

Цель stream: разгрузить PostgreSQL, убрать зависания на отчетах/импортах/reconciliation и привести SQL-слой к простому data access. SQL должен читать и писать данные, а бизнес-матчинг, обогащение справочниками, форматирование, сборка DTO и сложная валидация должны выполняться в Go.

## Общие правила stream

Рабочая папка:

```text
astra-crm-backend
```

Контекст:

```text
docs/domain-model.md
docs/csv-imports.md
docs/reconciliation-rules.md
docs/database-design.md
docs/testing-strategy.md
```

Правила:

- каждый изменяемый SQL-запрос должен быть покрыт тестом;
- каждый новый SQL-запрос должен быть проверен отдельно на типовом наборе данных;
- для hot queries нужно проверять план через `EXPLAIN` или `EXPLAIN ANALYZE` на локальной БД с seed/fixture данными;
- не запускать параллельные запросы через один `pgx.Tx`;
- независимые read-запросы запускать параллельно через goroutines, предпочтительно `errgroup.WithContext(ctx)`;
- если операция требует атомарности, сначала параллельно прочитать исходные данные через pool, затем открыть tx и выполнить write phase последовательно;
- не переносить из БД защиту целостности: `PRIMARY KEY`, `FOREIGN KEY`, `UNIQUE`, `CHECK`, partial unique indexes остаются в PostgreSQL;
- убрать функции и касты над колонками в hot `WHERE`/`JOIN`/`ORDER BY`, особенно `created_at_external::date`, `regexp_replace(...)`, `lower(...)`, `COALESCE(...)` по индексируемым колонкам;
- справочники, например `banks`, грузить отдельным запросом и обогащать в Go.

Done для всего stream:

- `go test ./...` проходит;
- API/UI контракт не меняется без отдельного решения;
- hot SQL не содержит бизнес-матчинга CSV/CRM;
- тяжелые CTE/FULL JOIN/jsonb_build_object вынесены из hot paths;
- импорт CSV не делает отдельный DB round-trip на каждую строку;
- добавлены индексы под реальные фильтры, JOIN и сортировки;
- тесты подтверждают matched/mismatch/csv_only/missing/payout-not-paid сценарии.

## SQL-OPT-001. Baseline SQL Tests and Query Inventory

Цель:

- зафиксировать текущее поведение перед оптимизацией;
- получить список hot queries и покрыть их regression-тестами.

Что делаем:

- составить inventory для query groups:
  - orders readmodels;
  - shift report;
  - trader reconciliation;
  - teamlead reconciliation;
  - imports;
  - requisites readmodels;
- добавить тестовые fixtures для:
  - active/superseded scope items;
  - inbound/outbound CSV;
  - shift requisites с verified/discrepancy/blocked;
  - manual payout orders и transfers;
  - banks;
- добавить тесты на текущее поведение:
  - фильтры по датам;
  - фильтры по trader/status/method/requisite;
  - сортировки;
  - empty result;
  - matched/mismatch reconciliation;
  - csv_only rows;
  - missing trader import;
  - payout not fully paid.

Done:

- есть regression tests для запросов, которые будут меняться в следующих задачах;
- тесты падают при изменении результата report/reconciliation/order lists;
- описан список запросов-кандидатов в task notes или комментариях к тестам.

## SQL-OPT-002. Add PostgreSQL Index Migration for Hot Paths

Цель:

- добавить недостающие индексы под реальные фильтры и сортировки до переписывания логики.

Что делаем:

- создать goose migration с индексами:
  - active `order_scope_items` по `team_id, shift_id, direction, created_at_external DESC, id DESC`;
  - active `order_scope_items` по `team_id, shift_id, direction, external_inner_id, created_at DESC, id DESC`;
  - active teamlead period scope items по `team_id, accounting_period_id, direction, external_inner_id, created_at DESC, id DESC`;
  - active teamlead current scope items по `team_id, import_batch_id, direction, external_inner_id, created_at DESC, id DESC`;
  - `shift_requisites` по `team_id, shift_id, taken_at DESC, id DESC`;
  - `shift_requisites` по `team_id, trader_id, shift_id, status`;
  - active/latest `requisite_assignments` по `team_id, requisite_id, assigned_for_date DESC, assigned_at DESC, id DESC`;
  - active `manual_payout_orders` по `team_id, trader_id, shift_id, id`;
  - `manual_payout_transfers` по `team_id, shift_id, trader_id, source_shift_requisite_id`;
  - recent teamlead current `import_batches` по `team_id, uploaded_by, direction, created_at DESC, id DESC`;
- проверить миграцию на fresh DB и existing DB;
- проверить, что planner использует индексы для hot queries после SQL-OPT-003/004.

Done:

- миграция применима и откатываема;
- индексы не дублируют существующие без причины;
- тесты миграций/репозитория проходят.

## SQL-OPT-003. Simplify Orders Readmodel Queries

Цель:

- сделать order list/dashboard queries простыми и индексируемыми.

Файлы:

```text
astra-crm-backend/sqlc/queries/readmodels.sql
astra-crm-backend/internal/orders
```

Что делаем:

- убрать `created_at_external::date` из `WHERE`;
- передавать из Go готовые `from timestamptz` и `toExclusive timestamptz`;
- заменить динамический `ORDER BY CASE` на простые варианты:
  - отдельные query для поддержанных сортировок;
  - либо единая дефолтная сортировка, если продуктово допустимо;
- убрать `count(*) OVER()` из list queries или заменить отдельным count-запросом;
- сохранить фильтры и response shape;
- распараллелить независимые запросы dashboard:
  - summary;
  - blocked balance;
  - status breakdown;
  - recent imports.

Тесты:

- date range не теряет крайние значения;
- default sort stable;
- amount sort работает, если остается в scope;
- filters по trader/status/method/requisite работают;
- pagination возвращает корректные элементы и total, если total остается в контракте;
- dashboard параллельные запросы возвращают тот же результат, что прежняя последовательная логика.

Done:

- hot `WHERE` не содержит кастов над колонками;
- readmodel tests покрывают фильтры и сортировки;
- независимые dashboard reads выполняются через goroutines/errgroup.

## SQL-OPT-004. Split Shift Report Query into Simple Reads

Цель:

- заменить тяжелый `ListShiftReportRows` на несколько простых запросов и сборку отчета в Go.

Файлы:

```text
astra-crm-backend/sqlc/queries/shifts.sql
astra-crm-backend/internal/shifts
```

Что делаем:

- вместо `ListShiftReportRows` добавить простые SQL-запросы:
  - get shift requisites by `team_id, shift_id`;
  - get active inbound scope items by `team_id, shift_id`;
  - get payout transfers by `team_id, shift_id`;
  - get latest inbound/outbound reconciliation;
- загрузку `banks` сделать отдельным запросом и обогащать строки в Go;
- перенести в Go:
  - нормализацию phone/card match keys;
  - сопоставление CSV к CRM;
  - подсчет inbound/outbound diffs;
  - `csv_only`;
  - сортировку report rows;
- запустить независимые reads параллельно через `errgroup.WithContext(ctx)`.

Тесты:

- report rows совпадают со старым поведением на matched case;
- `csv_only` rows появляются при CSV-реквизите без CRM match;
- phone/card matching работает на форматированных телефонах и картах;
- duplicate CRM match key не приводит к неправильному auto-match;
- mismatch sorting стабилен;
- banks enrichment корректен.

Done:

- в hot report SQL нет `regexp_replace`, `UNION`, `FULL JOIN`, многоступенчатых CTE;
- business matching живет в Go и покрыт unit tests;
- repository tests подтверждают итоговый report contract.

## SQL-OPT-005. Move Trader Reconciliation Item Generation to Go

Цель:

- вынести generation trader reconciliation items из SQL в Go.

Файлы:

```text
astra-crm-backend/sqlc/queries/reconciliation.sql
astra-crm-backend/internal/reconciliation
```

Что делаем:

- заменить тяжелые SQL:
  - `CountTraderInboundRequisiteMismatches`;
  - `CreateTraderInboundRequisiteMismatchItems`;
  - `UpdateTraderInboundRequisiteReviewStatuses`;
  - `CreateTraderOutboundReconciliationItems`;
- добавить простые read queries:
  - trader inbound active scope items;
  - closed/worked shift requisites;
  - manual payout orders;
  - manual payout transfers;
- перенести в Go:
  - phone/requisite matching;
  - inbound mismatch count;
  - status decision `worked_verified` / `worked_discrepancy`;
  - outbound payout matching by amount/match number;
  - JSON payload construction for `reconciliation_items`;
- write phase оставить в transaction:
  - create run;
  - insert prepared items;
  - update shift/requisite statuses;
  - update reconciliation status.

Тесты:

- trader inbound matched;
- trader inbound total mismatch;
- trader inbound requisite mismatch;
- trader inbound CSV-only/missing CRM;
- trader outbound matched;
- trader outbound missing manual payout;
- trader outbound extra manual payout;
- manual payout not fully paid;
- no concurrent usage of one `pgx.Tx` in implementation.

Done:

- trader reconciliation SQL не строит JSON и не делает business matching;
- все decisions покрыты unit tests;
- transaction boundaries очевидны.

## SQL-OPT-006. Move Teamlead Reconciliation Item Generation to Go

Цель:

- вынести teamlead current/period reconciliation item generation из SQL в Go.

Файлы:

```text
astra-crm-backend/sqlc/queries/reconciliation.sql
astra-crm-backend/internal/reconciliation
```

Что делаем:

- заменить тяжелые SQL:
  - `CreateTeamleadPeriodInboundReconciliationItems`;
  - `CreateTeamleadPeriodOutboundReconciliationItems`;
  - `CreateTeamleadCurrentReconciliationItems`;
- добавить простые read queries:
  - teamlead active period/current orders;
  - trader scope orders by external inner IDs;
  - shift requisites in accounting period;
  - manual payouts/transfers in accounting period;
- перенести в Go:
  - total comparison;
  - requisite-level comparison;
  - order-level comparison;
  - payout matching;
  - item message/value JSON construction.

Тесты:

- teamlead period inbound matched;
- teamlead period inbound total mismatch;
- teamlead period inbound requisite mismatch;
- teamlead period outbound matched;
- teamlead period outbound missing/extra payout;
- teamlead current inbound order missing/status/amount/worker mismatch;
- teamlead current outbound payout mismatch.

Done:

- teamlead reconciliation SQL состоит из simple reads и simple inserts/updates;
- reconciliation tests проходят без обращения к сложным SQL CTE;
- поведение API и UI не меняется.

## SQL-OPT-007. Remove Banks JOINs from Mass Read Queries

Цель:

- убрать справочник банков из массовых JOIN и обогащать данные в Go.

Что делаем:

- найти массовые queries с `JOIN banks`;
- оставить в SQL только `bank_code`;
- загрузить active banks отдельным read;
- добавить mapper/cache на уровне service/repository request scope;
- не менять API response shape: `bank_name` должен остаться в ответах, где он был.

Тесты:

- rows с известным `bank_code` получают `bank_name`;
- unknown/archived bank code обрабатывается безопасно;
- массовые list/report endpoints не теряют поле `bank_name`.

Done:

- в массовых report/list queries нет `JOIN banks`;
- справочник банков обогащается в Go;
- тесты покрывают enrichment.

## SQL-OPT-008. Batch CSV Import Database Writes

Цель:

- убрать построчные DB round-trips при CSV import.

Файлы:

```text
astra-crm-backend/sqlc/queries/imports.sql
astra-crm-backend/internal/imports
```

Что делаем:

- заменить построчный цикл:
  - `InsertImportRow`;
  - `UpsertExternalOrder`;
  - `CreateScopeItem`;
- на batch операции:
  - bulk insert `import_rows`;
  - bulk upsert `external_orders`;
  - bulk insert `order_scope_items`;
- сохранить reimport semantics:
  - old active scope items deactivate;
  - previous batches supersede;
  - latest active scope wins;
- не hard-delete historical imports;
- сохранить duplicate `innerId` validation до записи.

Тесты:

- import creates expected rows;
- reimport deactivates old scope items and activates new ones;
- duplicate `innerId` inside CSV rejected;
- external order upsert updates existing order;
- import with hundreds/thousands rows не делает per-row repository calls в тестируемом коде;
- transaction rollback leaves DB consistent on mid-import error.

Done:

- import path делает batch DB writes;
- результат ApplyImport совпадает с прежним контрактом;
- reimport/reconciliation side effects не сломаны.

## SQL-OPT-009. Raw Readmodels Cleanup

Цель:

- упростить raw SQL в `internal/readmodels/service.go`.

Что делаем:

- убрать `to_char` и построение UI labels из SQL;
- вернуть даты и суммы, форматировать title/date range в Go;
- убрать `created_at_external::date` и заменить на timestamptz ranges;
- разделить сложный `traderProfileQuery` на простые независимые reads;
- параллелить независимые reads через `errgroup.WithContext(ctx)`.

Тесты:

- `ListPeriods` возвращает прежние title/date range;
- `TraderProfile` сохраняет расчет salary;
- filter period работает с date_from/date_to;
- current period fallback работает.

Done:

- raw readmodel SQL не форматирует пользовательский текст;
- расчеты salary и period labels живут в Go и покрыты тестами.

## SQL-OPT-010. Final Verification and Performance Pass

Цель:

- убедиться, что stream достиг результата, а не только переписал код.

Что делаем:

- прогнать весь backend test suite;
- прогнать smoke scenarios импорта и reconciliation;
- собрать `EXPLAIN`/`EXPLAIN ANALYZE` для hot queries:
  - order lists;
  - dashboard summaries;
  - shift report source reads;
  - reconciliation source reads;
  - import deactivation/supersede queries;
- проверить отсутствие тяжелых SQL patterns в hot paths:
  - `created_at_external::date`;
  - `regexp_replace` в SQL;
  - `jsonb_build_object` в reconciliation item generation;
  - `FULL JOIN` для business matching;
  - `ORDER BY CASE` для UI sort.

Done:

- documented before/after notes по основным hot paths;
- `go test ./...` проходит;
- smoke import/reconciliation проходит;
- оставшиеся тяжелые SQL либо удалены, либо явно задокументированы как non-hot/acceptable.
