# Domain Model — p2p-crm

## 1. Overview

Главный доменный агрегат MVP: **TraderShift**.

Смена связывает:

- трейдера;
- реквизиты, взятые в работу;
- ежедневные данные по реквизитам;
- cumulative-обороты;
- ручные выплаты;
- intermediate transfers;
- CSV-импорты;
- reconciliation results;
- закрытие смены.

Второй важный агрегат: **ImportBatch**.

Импорт должен быть идемпотентным, переиспользуемым, историчным и поддерживать reimport с заменой активного набора данных.

---

## 2. Entity list

```text
Team
User
TraderProfile
Requisite
RequisiteAssignment
TraderShift
ShiftRequisite
RequisiteTurnoverEntry
ManualPayoutOrder
ManualPayoutTransfer
AccountingPeriod
ImportBatch
ImportRow
ExternalOrder
OrderScopeItem
ReconciliationRun
ReconciliationItem
AuditLog
AuthSession
```

---

# 3. Team

Команда тимлида/трейдеров.

Пока можно считать, что один тимлид управляет одной командой, но модель сразу должна иметь `team_id`, чтобы не переделывать систему позже.

Fields:

```text
id
name
status: active / archived
created_at
updated_at
```

Relationships:

```text
Team 1 -> N Users
Team 1 -> N Requisites
Team 1 -> N TraderShifts
Team 1 -> N AccountingPeriods
Team 1 -> N ExternalOrders
Team 1 -> N AuditLogs
```

---

# 4. User

Пользователь системы.

Roles:

```text
TEAMLEAD
TRADER
```

Fields:

```text
id
team_id
role
login
password_hash
status: active / disabled / deleted
created_at
updated_at
deleted_at nullable
```

Rules:

- Login must be unique globally or at least unique per team.
- Password hash must never be returned in API.
- Password hash must never be written to audit before/after payload.
- Physical deletion is forbidden once user has domain history.

---

# 5. TraderProfile

Дополнительные данные трейдера.

Fields:

```text
id
user_id
salary_rate_bps
external_worker_name
created_at
updated_at
```

Notes:

- `external_worker_name` maps CSV `workerName` to internal trader.
- Product decision: `workerName` is stable and unique for one trader.
- `salary_rate_bps` stores salary percent in basis points.

Examples:

```text
100 = 1%
50 = 0.5%
25 = 0.25%
```

Salary formula:

```text
salary_amount = successful_inbound_turnover * salary_rate_bps / 10000
```

---

# 6. Requisite

Base requisite created by teamlead.

Fields:

```text
id
team_id
phone
method_type: sbp / c2c / ...
proxy
status: active / disabled / archived
created_by
created_at
updated_at
deleted_at nullable
```

Important:

- Do not store daily card number / holder name here.
- Daily card/holder values belong to `ShiftRequisite`.
- Requisite can be assigned to a trader but can move between traders during day.
- Assignment history must be preserved.

---

# 7. RequisiteAssignment

History of assigning requisite to trader.

Fields:

```text
id
team_id
requisite_id
trader_id
assigned_by
assigned_at
unassigned_at nullable
comment nullable
```

Rules:

- One requisite can have only one active assignment.
- Active assignment means `unassigned_at is null`.
- Reassigning requisite closes previous active assignment and creates a new one.
- Assignment/reassignment must be audited.

Invariant:

```text
unique active requisite_id where unassigned_at is null
```

---

# 8. TraderShift

Trader work shift.

Fields:

```text
id
team_id
trader_id
started_at
ended_at nullable
status: open / closing / closed / closed_with_discrepancy
inbound_reconciliation_status: not_started / imported / matched / mismatch / accepted_with_comment
outbound_reconciliation_status: not_started / imported / matched / mismatch / accepted_with_comment
close_comment nullable
created_at
updated_at
closed_at nullable
```

Lifecycle:

```text
No open shift
  -> trader takes first assigned requisite into work
  -> system creates TraderShift(status=open)
  -> trader adds turnovers / payouts / imports
  -> system reconciles inbound/outbound
  -> trader closes shift
  -> status=closed or closed_with_discrepancy
```

Rules:

- Trader can have only one open shift at a time.
- Shift starts automatically when first requisite is taken into work.
- Shift cannot be closed until inbound and outbound reconciliations are matched or accepted with comment.
- Shift cannot be closed if any manual payout order is not fully paid.
- Closed shift cannot be reopened.

---

# 9. ShiftRequisite

A requisite taken into work during a specific shift.

Fields:

```text
id
team_id
shift_id
trader_id
requisite_id
assignment_id nullable
card_number
holder_name
taken_at
released_at nullable
status: active / released
created_at
updated_at
```

Meaning:

- This is the daily/shift-specific version of a requisite.
- Trader fills card number and holder name here.
- If for a method the actual daily instrument is phone rather than card, keep domain language flexible in UI, but persist in a stable field or rename later to `instrument_value`.

Rules:

- Trader can take into work only a requisite currently assigned to him.
- Taking first requisite creates current shift if absent.
- `card_number` and `holder_name` are required to take requisite into work.
- Changes should be audited.

---

# 10. RequisiteTurnoverEntry

Cumulative turnover entry for a shift requisite.

Fields:

```text
id
team_id
shift_id
shift_requisite_id
requisite_id
trader_id
amount
created_by
created_at
comment nullable
```

Rules:

- `amount` is cumulative, not delta.
- For reconciliation, take latest entry by `created_at` per `shift_requisite_id`.
- Entries are append-only in MVP. If editing is allowed later, audit it heavily.

Example:

```text
12:00, Requisite A, amount = 50_000
15:00, Requisite A, amount = 120_000
18:00, Requisite A, amount = 180_000
```

For shift reconciliation, Requisite A contributes `180_000`.

---

# 11. ManualPayoutOrder

Manual payout order created by trader.

Fields:

```text
id
team_id
shift_id
trader_id
destination_bank
destination_requisite
amount
status: draft / in_progress / paid / cancelled
created_at
updated_at
deleted_at nullable
```

Rules:

- Payout order can be split into multiple intermediate transfers.
- Payout order is `paid` only when sum of transfers equals `amount`.
- Sum of transfers must not exceed `amount`.
- Shift cannot close with unpaid manual payout order.

---

# 12. ManualPayoutTransfer

Intermediate transfer that fills a manual payout order.

Fields:

```text
id
team_id
manual_payout_order_id
shift_id
trader_id
source_shift_requisite_id
source_requisite_id
amount
created_by
created_at
comment nullable
```

Example:

```text
Payout order: 15_000
Transfer 1 from Requisite A: 5_000
Transfer 2 from Requisite B: 5_000
Transfer 3 from Requisite C: 5_000
Remaining: 0
```

Rules:

- `source_shift_requisite_id` is preferred because payout happened from a specific shift requisite, not just a base requisite.
- Outbound reconciliation uses `sum(manual_payout_transfers.amount)`.

---

# 13. AccountingPeriod

Period controlled by teamlead, usually month/30 days.

Fields:

```text
id
team_id
date_from
date_to
status: open / checking / closed / closed_with_discrepancy
created_by
created_at
closed_by nullable
closed_at nullable
```

Rules:

- Teamlead imports CSV for a period.
- Teamlead inbound period reconciliation compares teamlead CSV vs trader shift imports.
- Teamlead outbound period import exists in MVP, but no advanced triggers yet.

---

# 14. ImportBatch

Fact of CSV upload.

Fields:

```text
id
team_id
uploaded_by
scope_type: trader_shift / teamlead_period
direction: inbound / outbound
shift_id nullable
accounting_period_id nullable
trader_id nullable
file_name
file_hash
rows_count
status: uploaded / parsed / applied / reconciled / superseded / failed
superseded_by_batch_id nullable
error_message nullable
created_at
applied_at nullable
```

Scope keys:

```text
Trader shift import scope:
  scope_type = trader_shift
  shift_id
  direction

Teamlead period import scope:
  scope_type = teamlead_period
  accounting_period_id
  direction
```

Reimport rule:

- New import in same scope supersedes previous active import.
- Old import remains historical.
- New active order scope set fully replaces previous active scope set.
- Reimport triggers reconciliation recalculation.

---

# 15. ImportRow

Raw CSV row.

Fields:

```text
id
import_batch_id
row_number
external_id
external_inner_id
raw_payload_json
parse_status: parsed / failed
parse_error nullable
created_at
```

Rules:

- Store the raw payload even after normalization.
- This enables debugging disputes and changing parser logic later.
- Row numbers should match CSV row positions for user-facing import errors.

---

# 16. ExternalOrder

Normalized external order from CSV.

Fields:

```text
id
team_id
direction: inbound / outbound
external_id
external_inner_id
external_foreign_id nullable
worker_name
trader_id nullable
requisite_raw
requisite_phone nullable
requisite_external_id nullable
requisite_id nullable
device_name
method_type
method_name
amount
currency
course nullable
course_worker nullable
worker_amount nullable
worker_profit nullable
raw_status
normalized_status: success / corrected / failed / cancelled / unknown
created_at_external
closed_at_external nullable
updated_at_external nullable
old_amount nullable
had_dispute nullable
receipt nullable
order_comment nullable
ordered nullable
counted nullable
initials nullable
last_import_batch_id
created_at
updated_at
```

Uniqueness:

```text
unique(team_id, direction, external_inner_id)
```

Notes:

- `innerId` is the main external unique key.
- `id` from CSV should also be stored as `external_id`.
- Store fields that are currently not used (`workerAmount`, `workerProfit`, `ordered`, `counted`, `initials`) for future logic.

---

# 17. OrderScopeItem

Active/inactive link between normalized order and an import scope.

Fields:

```text
id
team_id
scope_type: trader_shift / teamlead_period
direction: inbound / outbound
shift_id nullable
accounting_period_id nullable
import_batch_id
import_row_id
external_order_id
is_active
created_at
deactivated_at nullable
```

Why needed:

- Same external order can appear in trader CSV and teamlead CSV.
- Same scope can be reimported many times.
- We need current calculations to use only the latest active scope set.
- We need historical import data for audit/debug.

Reimport algorithm:

```text
1. Start transaction.
2. Create new import_batch.
3. Parse and validate CSV.
4. If duplicate innerId inside file -> fail import.
5. Save import_rows.
6. Upsert external_orders by team_id + direction + external_inner_id.
7. Deactivate previous active order_scope_items for same scope.
8. Mark previous active import_batch for same scope as superseded.
9. Create new active order_scope_items.
10. Run reconciliation.
11. Write audit event.
12. Commit.
```

---

# 18. ReconciliationRun

Result of a reconciliation calculation.

Fields:

```text
id
team_id
type: trader_shift_inbound / trader_shift_outbound / teamlead_period_inbound / teamlead_period_outbound
scope_type: trader_shift / teamlead_period
shift_id nullable
accounting_period_id nullable
trader_id nullable
import_batch_id nullable
expected_amount
actual_amount
diff_amount
success_amount
success_count
failed_amount
failed_count
total_amount
total_count
status: matched / mismatch / accepted_with_comment
comment nullable
confirmed_by nullable
confirmed_at nullable
created_at
```

Meaning of expected/actual depends on type. See `docs/reconciliation-rules.md`.

---

# 19. ReconciliationItem

Detailed mismatch line.

Fields:

```text
id
reconciliation_run_id
issue_type: missing_in_trader_import / extra_in_trader_import / amount_mismatch / status_mismatch / worker_mismatch / total_amount_mismatch / payout_not_fully_paid
external_order_id nullable
external_inner_id nullable
teamlead_value_json nullable
trader_value_json nullable
message nullable
created_at
```

Used mostly for teamlead period reconciliation and detailed UI.

---

# 20. AuditLog

Append-only audit event.

Fields:

```text
id
team_id
actor_id
action
entity_type
entity_id
before_json nullable
after_json nullable
changed_fields_json nullable
comment nullable
created_at
```

Rules:

- Audit all mutations.
- Audit import/reimport results.
- Audit mismatch acceptance and comments.
- Do not audit secrets/password/hash/token values.
- Prefer redacting sensitive fields in `before_json` / `after_json`.

---

# 21. AuthSession

Server-side user session.

Fields:

```text
id
user_id
token_hash
user_agent nullable
ip nullable
expires_at
created_at
revoked_at nullable
```

Rules:

- Store only token hash.
- Cookie stores raw token.
- Logout sets `revoked_at`.
- Middleware validates session and user status.

---

# 22. Status summary

## TraderShift.status

```text
open
closing
closed
closed_with_discrepancy
```

## Reconciliation status

```text
not_started
imported
matched
mismatch
accepted_with_comment
```

## ImportBatch.status

```text
uploaded
parsed
applied
reconciled
superseded
failed
```

## ExternalOrder.normalized_status

```text
success
corrected
failed
cancelled
unknown
```

## ManualPayoutOrder.status

```text
draft
in_progress
paid
cancelled
```

---

# 23. Core invariants

```text
A trader can have only one open shift.
A requisite can have only one active assignment.
A trader can take only assigned requisites into work.
Taking first requisite creates shift automatically.
Turnover entries are cumulative.
Reconciliation uses latest turnover entry per shift requisite.
CSV duplicate innerId inside one file is an import error.
CSV reimport replaces active order scope set.
Historical import batches/rows are preserved.
Inbound success includes hand_success and corrected.
Outbound reconciliation compares CSV amount vs manual payout transfers.
Manual payout transfer sum cannot exceed payout order amount.
Payout order is paid only when transfer sum equals amount.
Shift cannot close if payout orders are incomplete.
Shift cannot close without inbound and outbound reconciliation.
Any mismatch requires comment.
Closed shift cannot be reopened.
Deletes are soft deletes for domain entities with history.
All edits are audited.
```
