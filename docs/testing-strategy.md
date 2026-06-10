# Testing Strategy

## 1. Priorities

The most important test areas:

1. CSV parser.
2. CSV reimport semantics.
3. Reconciliation calculations.
4. Shift closing rules.
5. Manual payout transfer rules.
6. Authorization/RBAC.
7. Audit for mutations.

---

## 2. Backend unit tests

### CSV parser tests

Use example files:

```text
teamlead-1780661585.csv
traider-1780661666.csv
```

Test:

```text
detect separator |
parse teamlead columns
parse trader columns
parse optional fields
parse None as null
parse dates DD.MM.YYYY HH:mm:ss
parse amount without float
map statuses
count rows
count statuses
reject missing required columns
reject duplicate innerId inside one file
```

### Status mapping tests

```text
hand_success -> success group
corrected -> success group
auto_decline -> failed group
cancelled -> failed group
unknown -> unknown
```

### Money parsing tests

```text
3001.0 RUB -> 300100 minor units
3068.0 RUB -> 306800 minor units
0 -> 0
invalid decimal -> error
```

---

## 3. Reconciliation tests

### Trader inbound

Test:

```text
matched when CSV success amount equals latest cumulative turnovers
mismatch when diff exists
corrected counted as success
latest turnover per shift requisite is used
older turnover entries ignored
accepted mismatch requires comment
empty comment rejected
```

### Trader outbound

Test:

```text
matched when CSV payout amount equals sum transfers
mismatch when diff exists
payout cannot be paid until transfer sum equals amount
transfer cannot exceed remaining amount
shift cannot close with unpaid payout
accepted mismatch requires comment
```

### Teamlead period inbound

Test:

```text
matched when teamlead CSV equals trader active imports
missing_in_trader_import detected
extra_in_trader_import detected
amount_mismatch detected
status_mismatch detected
worker_mismatch detected
corrected counted as success
```

---

## 4. Reimport tests

Important cases:

```text
first import creates batch, rows, external_orders, active scope items
second import same scope marks previous batch superseded
second import deactivates previous scope items
second import creates new active scope items
old historical rows remain
order removed from new file is inactive in scope
order changed in new file updates external_orders
reconciliation reruns after reimport
```

Example:

```text
Old CSV: A, B, C
New CSV: A, C, D
Active after reimport: A, C, D
B inactive for this scope
```

---

## 5. Shift lifecycle tests

Test:

```text
taking first requisite creates shift
taking second requisite uses existing open shift
trader can have only one open shift
trader cannot take unassigned requisite
shift cannot close without inbound import
shift cannot close without outbound import
shift cannot close with unresolved mismatch
shift can close with accepted mismatch and comment
closed shift cannot be reopened
```

---

## 6. Payout tests

Test:

```text
create payout order
add transfer below remaining
add transfer equal remaining -> payout paid
add transfer above remaining -> error
cancel payout audited
outbound reconciliation uses transfers not payout headers
```

---

## 7. Audit tests

For each mutation, assert audit exists:

```text
create trader
reset password
create requisite
assign requisite
take requisite
add turnover
create payout
add transfer
import CSV
accept mismatch
close shift
```

Assert sensitive fields absent:

```text
password
password_hash
session token
```

---

## 8. Integration tests

Use real PostgreSQL in tests.

Possible tools:

- docker-compose test DB;
- testcontainers-go;
- local Postgres in CI.

Integration tests should cover:

```text
repository queries
transactions
partial unique indexes
reimport transaction
close shift transaction
```

---

## 9. API tests

Test role access:

```text
trader cannot access teamlead endpoints
trader cannot view another trader shift
teamlead can view team data
unauthenticated request rejected
```

Test endpoints:

```text
login/logout/me
create trader
create requisite
assign requisite
take requisite
add turnover
import inbound CSV
create payout/add transfer
import outbound CSV
close shift
```

---

## 10. Frontend tests

MVP can be light, but cover critical forms:

```text
login form validation
requisite form validation
trader form validation
turnover form explains cumulative amount
payout transfer cannot exceed remaining
mismatch comment required
close shift checklist disables button when blocked
```

Use:

```text
Vitest
React Testing Library
Playwright later for e2e
```

---

## 11. Golden files

Keep sanitized CSV examples under:

```text
/testdata/csv/teamlead.csv
/testdata/csv/trader.csv
```

If original files contain sensitive data, create sanitized versions with same structure.

Golden tests should not depend on private real-world data unless repo is private and access-controlled.
