# Backend SQL Optimization Notes

Дата: 2026-06-29

## Что изменено

- Orders readmodels: убраны `created_at_external::date`, `count(*) OVER()` и динамический `ORDER BY CASE`; list/count разделены.
- Shift report: тяжелая сборка отчета вынесена из SQL в Go, SQL оставлен как набор простых source reads.
- Trader/teamlead reconciliation: генерация item'ов и JSON payload перенесена в Go; SQL читает источники и делает простые insert/update.
- Banks enrichment: `JOIN banks` убран из массовых list/report queries; `bank_name` обогащается в Go через active banks lookup.
- CSV import: post-row write loop заменен batch insert/upsert/scope insert через `jsonb_to_recordset`.
- Raw readmodels: period labels, date ranges и salary считаются в Go; timestamp filters используют half-open `timestamptz` ranges.

## Проверка

- `go test ./...`: 234 теста прошли.
- API smoke `scripts/smoke-api.mjs`: прошел. Сценарий покрывает trader inbound/outbound import, reconciliation, accept mismatch и close shift.
- Static SQL scan:
  - `created_at_external::date`: не найден в production SQL по hot paths.
  - `JOIN banks`: не найден в sqlc queries.
  - `jsonb_build_object`, `FULL JOIN`, `count(*) OVER()`: не найдены в hot query paths.

## EXPLAIN на локальной compose DB

Representative `EXPLAIN (COSTS OFF)`:

- Order list by shift/direction/date uses `idx_order_scope_shift_active`.
- External orders period lookup uses `idx_external_orders_created_external`.
- Scope deactivation update uses `idx_order_scope_shift_active`.
- Shift requisites source read picked sequential scan on tiny local dataset; query shape matches index `idx_shift_requisites_team_shift_taken`.

## Оставшиеся known issues

- `regexp_replace(...)` остался только в пользовательском поиске requisites/shifts по digits. Это не reconciliation/report business matching, но для больших таблиц лучше заменить отдельными normalized/search columns и индексами.
- Integration rollback test для `ApplyImport` требует real PostgreSQL test harness. Сейчас покрыто unit/baseline тестами и API smoke; отдельный Go integration harness описан в `docs/testing-strategy.md`, но еще не реализован.
