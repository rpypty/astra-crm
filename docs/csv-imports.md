# CSV Imports

## 1. General

CSV import is one of the core parts of p2p-crm.

Both teamlead and trader import CSV files from an external admin system.

Product decision:

- Teamlead and trader can upload CSV multiple times.
- Reimport must fully replace the active set for the same scope.
- Reimport must trigger recalculation of orders and reconciliation.
- Historical imports and raw rows must be preserved.

---

## 2. CSV separator

Observed separator:

```text
|
```

Parser must not assume comma-separated CSV.

---

## 3. Main identifiers

Observed identifiers:

```text
id
foreignId
innerId
```

Main external unique key:

```text
innerId
```

Recommended uniqueness in DB:

```text
unique(team_id, direction, external_inner_id)
```

Store also:

```text
external_id = CSV id
external_foreign_id = CSV foreignId when present
```

---

## 4. Observed teamlead CSV

Example file name:

```text
teamlead-1780661585.csv
```

Observed stats from example:

```text
Rows: 9206
Columns: 22
```

Columns:

```text
id
foreignId
innerId
requisite
requisitePhone
deviceName
methodType
methodName
amount
course
courseWorker
currency
status
createdAt
closedAt
updatedAt
oldAmount
hadDispute
receipt
orderComment
workerName
workerAmount
```

Observed statuses:

```text
hand_success: 5288
auto_decline: 3909
cancelled: 7
corrected: 2
```

Observed method types:

```text
СБП
C2C
```

Observed worker names include:

```text
Bliss_OP8
Bliss_OP2
Bliss_OP3
Bliss_OP1
Bliss_OP9
Bliss_OP6
Bliss_OP11
Bliss_OP14
Bliss_OP5
Bliss_OP12
```

---

## 5. Observed trader CSV

Example file name:

```text
traider-1780661666.csv
```

Observed stats from example:

```text
Rows: 1145
Columns: 24
```

Columns:

```text
id
innerId
requisite
requisitePhone
deviceName
methodType
methodName
amount
courseWorker
currency
status
createdAt
closedAt
updatedAt
oldAmount
receipt
orderComment
requisiteId
workerName
workerAmount
workerProfit
ordered
counted
initials
```

Observed statuses:

```text
hand_success: 609
auto_decline: 536
```

Observed method types:

```text
СБП
C2C
```

Observed worker name:

```text
Bliss_OP2
```

Important observation:

- Trader CSV is a subset/slice of teamlead CSV for one `workerName`.
- In the example, all trader orders are present in teamlead file by `id`/`innerId`.
- Therefore, use one normalized order model with different import scopes.

---

## 6. Columns that may differ by role

Teamlead file has:

```text
foreignId
course
hadDispute
```

Trader file has:

```text
requisiteId
workerProfit
ordered
counted
initials
```

Importer should handle optional columns.

Recommended normalized fields:

```text
external_id
external_inner_id
external_foreign_id nullable
requisite_raw
requisite_phone nullable
requisite_external_id nullable
device_name
method_type
method_name
amount_minor
course nullable
course_worker nullable
currency
raw_status
normalized_status
created_at_external
closed_at_external nullable
updated_at_external nullable
old_amount_minor nullable
had_dispute nullable
receipt nullable
order_comment nullable
worker_name
trader_id nullable
worker_amount nullable
worker_profit nullable
ordered nullable
counted nullable
initials nullable
```

---

## 7. Status mapping

Raw -> normalized:

```text
hand_success -> success
corrected    -> corrected
...
```

Business grouping:

```text
success group: hand_success, corrected
failed group: auto_decline, cancelled
```

`corrected` is counted as successful turnover.

Unknown statuses:

- Store raw status.
- Set normalized status `unknown`.
- Show warning in import result.
- Do not silently count as success.

---

## 8. Date parsing

Observed format:

```text
DD.MM.YYYY HH:mm:ss
```

Examples:

```text
28.05.2026 11:21:35
23.05.2026 21:13:38
```

Special value:

```text
None
```

Parser must convert `None` to null.

Recommended fields:

```text
created_at_external timestamptz or timestamp
closed_at_external nullable
updated_at_external nullable
```

Timezone needs product decision. If external admin uses a known timezone, document it. Default assumption for MVP can be Europe/Moscow or server-configured timezone, but avoid silently using local machine timezone.

---

## 9. Money parsing

Observed examples:

```text
3068.0
3001.0
```

Do not parse via float.

Recommended:

- Parse decimal string exactly.
- Store money as `amount_minor bigint`.
- For RUB, `3001.0` -> `300100` minor units.

If product decides rubles-only with no kopeks, can store `amount bigint`, but default safe choice is minor units.

---

## 10. Import scopes

Trader inbound shift import:

```text
scope_type = trader_shift
shift_id = <current shift>
direction = inbound
```

Trader outbound shift import:

```text
scope_type = trader_shift
shift_id = <current shift>
direction = outbound
```

Teamlead inbound period import:

```text
scope_type = teamlead_period
accounting_period_id = <period>
direction = inbound
```

Teamlead outbound period import:

```text
scope_type = teamlead_period
accounting_period_id = <period>
direction = outbound
```

---

## 11. Reimport behavior

When uploading a new file into a scope that already has active import data:

```text
Old active order_scope_items -> inactive
Old active import_batch -> superseded
New import_batch -> active/applied/reconciled
New order_scope_items -> active
Reconciliation -> recalculated
Audit -> import.reuploaded
```

Example:

```text
Old active CSV: A, B, C
New CSV: A, C, D
```

After reimport, active set:

```text
A, C, D
```

`B` remains historical but inactive for that scope.

If `A` or `C` changed fields, update normalized `external_orders` with latest values.

---

## 12. Duplicate innerId inside one file

Recommended MVP rule:

```text
Reject entire import if same innerId appears more than once inside the same CSV.
```

Why:

- Financial reconciliation should not silently resolve duplicates.
- User should receive clear error with duplicated `innerId` and row numbers.

Potential future alternative:

- Last row wins inside file.
- But do not implement unless explicitly requested.

---

## 13. Import result response

After import, API should return a result summary:

```json
{
  "importBatchId": "...",
  "status": "reconciled",
  "rowsCount": 1145,
  "createdOrders": 10,
  "updatedOrders": 1135,
  "deactivatedScopeItems": 1145,
  "activeScopeItems": 1145,
  "unknownStatuses": [],
  "reconciliation": {
    "status": "matched",
    "expectedAmountMinor": 125000000,
    "actualAmountMinor": 125000000,
    "diffAmountMinor": 0
  }
}
```

For error:

```json
{
  "status": "failed",
  "errorCode": "CSV_DUPLICATE_INNER_ID",
  "message": "CSV contains duplicated innerId values",
  "details": [
    {
      "innerId": "...",
      "rows": [12, 40]
    }
  ]
}
```

---

## 14. Import UX

Import modal should show:

- accepted format;
- max file size;
- expected separator `|`;
- drag & drop area;
- selected file name;
- warning on reimport: “Этот импорт заменит текущие активные данные по смене/периоду”.

Import result dialog should show:

- success/fail state;
- rows count;
- orders count;
- status breakdown;
- successful amount;
- failed amount;
- expected vs actual amount;
- diff;
- list of errors/warnings if any.

Mismatch dialog should show:

- red alert;
- expected amount;
- actual amount;
- diff;
- primary call to action: “Исправить данные”;
- secondary destructive/explicit action: “Подтвердить с комментарием”.

---

## 15. Golden tests

Add tests using example CSVs:

```text
teamlead-1780661585.csv
traider-1780661666.csv
```

Test cases:

```text
parse teamlead columns
parse trader columns
detect separator |
parse None as null
parse money exactly
map statuses
count rows
count statuses
upsert orders by innerId
trader CSV orders are valid normalized orders
reimport replaces active scope set
duplicate innerId inside same file fails import
```
