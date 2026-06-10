# Reconciliation Rules

## 1. Core idea

Reconciliation is the heart of p2p-crm.

The system compares:

- CSV data from external admin;
- manual/cumulative data entered by traders;
- trader shift imports;
- teamlead period imports.

Every reimport must recalculate reconciliation.

---

# 2. Status mapping

Raw CSV statuses observed:

```text
hand_success
auto_decline
cancelled
corrected
```

Mapping:

```text
hand_success -> success
corrected    -> corrected, counted as success
ignore/unknown statuses -> unknown
```

For dashboards:

```text
Success statuses: hand_success, corrected
Failure statuses: auto_decline, cancelled
Corrected: keep separately visible, but include in success amount/count
Unknown: show separately and do not silently count as success
```

---

# 3. Trader inbound reconciliation

Triggered when trader imports inbound CSV for current shift, and again on reimport or turnover changes.

Scope:

```text
scope_type = trader_shift
shift_id = current shift
direction = inbound
```

Expected amount:

```text
expected_amount = sum(active CSV orders amount where raw_status in hand_success, corrected)
```

Actual amount:

```text
actual_amount = sum(latest cumulative turnover amount per shift_requisite)
```

Difference:

```text
diff_amount = actual_amount - expected_amount
```

Matched:

```text
diff_amount == 0
```

Mismatch:

```text
diff_amount != 0
```

If mismatch:

- show red alert;
- show expected, actual, diff;
- trader must either correct data or accept mismatch with required comment;
- accepted mismatch sets reconciliation status `accepted_with_comment`.

Shift effect:

```text
matched -> shift.inbound_reconciliation_status = matched
mismatch -> shift.inbound_reconciliation_status = mismatch
accepted_with_comment -> shift.inbound_reconciliation_status = accepted_with_comment
```

---

# 4. Trader outbound reconciliation

Triggered when trader imports outbound CSV for current shift, and again on reimport or manual payout changes.

Scope:

```text
scope_type = trader_shift
shift_id = current shift
direction = outbound
```

Expected amount:

```text
expected_amount = sum(active payout CSV order amounts)
```

Actual amount:

```text
actual_amount = sum(manual_payout_transfers.amount for shift)
```

Difference:

```text
diff_amount = actual_amount - expected_amount
```

Important: actual amount is based on intermediate transfers, not manual payout order headers.

Matched:

```text
diff_amount == 0
```

Mismatch:

```text
diff_amount != 0
```

Additional hard rule:

```text
All manual_payout_orders for the shift must be fully paid.
```

A manual payout order is fully paid if:

```text
sum(manual_payout_transfers.amount) == manual_payout_order.amount
```

If any payout order has remaining amount:

- shift cannot be closed;
- UI must show which payout is incomplete;
- reconciliation item may be `payout_not_fully_paid`.

---

# 5. Shift closing rules

A shift can be closed only if all conditions are true:

```text
1. inbound CSV was imported;
2. inbound reconciliation status is matched or accepted_with_comment;
3. outbound CSV was imported;
4. outbound reconciliation status is matched or accepted_with_comment;
5. all manual payout orders are fully paid;
6. any accepted mismatch has non-empty required comment.
```

Final shift status:

```text
closed
```

if both inbound/outbound are matched.

```text
closed_with_discrepancy
```

if at least one of inbound/outbound is accepted_with_comment.

Closed shift cannot be reopened.

---

# 6. Teamlead period inbound reconciliation

Triggered when teamlead imports inbound CSV for accounting period, and again on reimport or related trader import changes.

Scope:

```text
scope_type = teamlead_period
accounting_period_id = period
direction = inbound
```

Comparison source A:

```text
teamlead active period CSV orders
```

Comparison source B:

```text
active trader shift inbound CSV orders for shifts within the period
```

The period boundary should use external order timestamps and/or shift timestamps. MVP should be explicit and consistent. Recommended:

```text
Use order.created_at_external for external orders and shift.started_at/closed_at for shift grouping.
```

Levels of reconciliation:

## Level 1: total success amount/count

```text
teamlead_success_amount = sum(teamlead active period orders where success/corrected)
trader_success_amount = sum(active trader shift inbound orders where success/corrected)
```

Compare:

```text
teamlead_success_amount vs trader_success_amount
teamlead_success_count vs trader_success_count
```

## Level 2: by workerName/trader

Group by `workerName` / mapped `trader_id`.

Compare:

```text
amount by trader
count by trader
```

## Level 3: by order innerId

Compare by `external_inner_id`.

Issue types:

```text
missing_in_trader_import
extra_in_trader_import
amount_mismatch
status_mismatch
worker_mismatch
```

Examples:

```text
Order exists in teamlead CSV but not in any trader shift CSV -> missing_in_trader_import
Order exists in trader CSV but not in teamlead CSV -> extra_in_trader_import
Same innerId but different amount -> amount_mismatch
Same innerId but different normalized status -> status_mismatch
Same innerId but different workerName -> worker_mismatch
```

Status:

```text
matched -> no issues and totals match
mismatch -> issues exist
accepted_with_comment -> teamlead accepted with required comment
```

---

# 7. Teamlead period outbound import

MVP rule:

- Teamlead can import outbound CSV for period.
- Store it as `ImportBatch` + `ImportRows` + `ExternalOrders` + active `OrderScopeItems`.
- No advanced reconciliation triggers yet unless explicitly requested later.

The model supports future `teamlead_period_outbound` reconciliation.

---

# 8. Reimport recalculation rules

When a CSV is uploaded for a scope that already has active import data:

```text
1. New import replaces active set for that scope.
2. Previous import is marked superseded.
3. Previous order_scope_items for scope become inactive.
4. New order_scope_items become active.
5. ExternalOrder records are upserted by team_id + direction + innerId.
6. Reconciliation for the affected scope is recalculated.
7. Parent shift/period statuses are updated.
8. Audit event is written.
```

Important behavior:

```text
Old file: A, B, C
New file: A, C, D
Active set after reimport: A, C, D
B remains historical but inactive for that scope.
```

---

# 9. Amount precision

CSV amounts may come like:

```text
3001.0
3068.0
```

Domain should store money as integer minor units or integer rubles. MVP decision needed at implementation time:

Option A:

```text
amount_minor bigint
3001.0 RUB -> 300100 minor units
```

Option B:

```text
amount bigint
3001.0 RUB -> 3001
```

Recommended for safety: store `amount_minor bigint`, parse decimals exactly.

Never use float for persisted money or reconciliation.

---

# 10. Salary calculation

For trader shift dashboard:

```text
successful_inbound_turnover = sum(active inbound CSV order amount where status in hand_success, corrected)
salary = successful_inbound_turnover * trader.salary_rate_bps / 10000
```

If using minor units:

```text
salary_minor = successful_inbound_turnover_minor * salary_rate_bps / 10000
```

Potential issue: rounding. MVP should define rounding. Recommended:

```text
Use integer division floor for MVP, display rounded to currency format.
```

---

# 11. Dashboard formulas

Inbound dashboard:

```text
success_amount = sum(amount where status in hand_success, corrected)
success_count = count(status in hand_success, corrected)
failed_amount = sum(amount where status in auto_decline, cancelled)
failed_count = count(status in auto_decline, cancelled)
total_amount = sum(all known statuses)
total_count = count(all orders)
conversion_by_count = success_count / total_count
conversion_by_amount = success_amount / total_amount
```

Outbound dashboard:

```text
csv_payout_amount = sum(active outbound CSV amount)
manual_transfer_amount = sum(manual_payout_transfers.amount)
diff_amount = manual_transfer_amount - csv_payout_amount
manual_orders_total = count(manual_payout_orders)
manual_orders_paid = count(status = paid)
manual_orders_unpaid = count(status != paid and status != cancelled)
```
