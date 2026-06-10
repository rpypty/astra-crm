# Security and Audit

## 1. Authentication

MVP auth:

```text
login/password
server-side sessions
httpOnly cookie
```

Password storage:

- Store password hash only.
- Use a modern password hashing algorithm such as bcrypt or argon2id.
- Never log plaintext password.
- Never return password hash in API.
- Never store password hash in audit before/after JSON.

Session storage:

- Store session token hash in DB.
- Cookie contains raw session token.
- Session can be revoked.
- Session has expiration.

---

## 2. Authorization

Roles:

```text
TEAMLEAD
TRADER
```

Authorization must be enforced on backend, not only frontend.

TEAMLEAD can:

- manage traders within team;
- manage requisites within team;
- view team orders/imports/reconciliations;
- import teamlead period CSV;
- view audit.

TRADER can:

- view only own assigned requisites;
- manage only own current shift data;
- import only own shift CSV;
- manage own manual payouts;
- close own shift.

TRADER cannot:

- view other traders;
- manage assignments;
- view global audit;
- import teamlead period CSV.

---

## 3. Multi-team data isolation

All domain tables should include `team_id` where applicable.

Every query should filter by `team_id` derived from authenticated user/session.

Never trust `team_id` from client request if it can be inferred from session.

---

## 4. Audit principles

Audit log is append-only.

Audit all mutations:

```text
user.created
user.updated
user.password_reset
user.disabled

requisite.created
requisite.updated
requisite.archived
requisite.assigned
requisite.unassigned
requisite.reassigned

shift.created
shift.requisite_taken
shift.daily_details_updated
shift.turnover_added
shift.closed
shift.closed_with_discrepancy

manual_payout.created
manual_payout.updated
manual_payout.cancelled
manual_payout.transfer_added
manual_payout.transfer_deleted

import.uploaded
import.applied
import.reuploaded
import.failed

reconciliation.created
reconciliation.accepted_with_comment

period.created
period.closed
period.closed_with_discrepancy
```

Audit event fields:

```text
actor_id
action
entity_type
entity_id
before_json
after_json
changed_fields_json
comment
created_at
```

---

## 5. Redaction

Never audit/log:

- plaintext password;
- password hash;
- session token;
- session token hash;
- secrets;
- raw cookie values.

Potentially sensitive values such as card number can be audited carefully depending on product requirement. If auditing card number changes, consider masking:

```text
**** **** **** 1234
```

For debugging financial issues, full values may be operationally useful, but this increases risk. Decide deliberately.

Recommended MVP:

- Store full card number in domain table if required for operation.
- In audit `changed_fields_json`, mask card number.

---

## 6. CSV upload security

CSV upload rules:

- Check content type loosely but do not rely on it.
- Limit file size.
- Limit row count if needed.
- Parse as text with expected encoding.
- Reject malformed rows.
- Reject duplicate `innerId` within file.
- Store raw payload but do not execute anything from file.
- Escape values in UI to prevent injection.

CSV formula injection:

If exported later, sanitize cells starting with:

```text
=
+
-
@
```

---

## 7. Error handling

User-facing errors should be safe and clear.

Good:

```text
CSV содержит дубликаты innerId. Проверьте строки 12 и 40.
```

Bad:

```text
pq: duplicate key violates unique constraint uq_external_orders
```

Log technical details server-side with request_id.

---

## 8. Request logging

Log:

- request_id;
- method;
- path;
- status;
- latency;
- user_id if authenticated;
- team_id if authenticated.

Do not log:

- request bodies containing passwords;
- cookies;
- full uploaded CSV content.

---

## 9. Destructive actions

Use soft delete/archive for:

- users/traders;
- requisites;
- manual payout orders where history exists.

Hard delete only for safe transient data, if any.

UI must confirm:

- archive requisite;
- disable trader;
- delete/cancel payout;
- delete transfer;
- accept mismatch;
- close shift with discrepancy;
- reimport CSV replacing active scope.

---

## 10. CSRF

Because auth uses cookie sessions, add CSRF protection or SameSite strategy.

Recommended:

- `SameSite=Lax` or `Strict` where possible.
- For unsafe methods, CSRF token header if needed.

---

## 11. Rate limiting

MVP basic rate limits:

- login endpoint;
- CSV import endpoint.

Can be simple in-memory per instance for MVP, but for distributed deploy use Redis/Postgres-backed limiter.
