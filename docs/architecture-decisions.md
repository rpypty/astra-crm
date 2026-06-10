# Architecture Decisions

## ADR-001: Modular monolith first

Decision:

```text
Use Go modular monolith for MVP.
```

Why:

- Domain is tightly connected: shifts, requisites, imports, payouts, reconciliation.
- Microservices would add operational complexity too early.
- Modular monolith keeps boundaries clear while preserving speed.
- Can extract services later if specific modules grow.

Modules:

```text
auth
users
teams
requisites
shifts
imports
orders
payouts
reconciliation
audit
dashboard
```

---

## ADR-002: PostgreSQL as main database

Decision:

```text
Use PostgreSQL.
```

Why:

- Financial reconciliation needs transactions.
- Strong constraints and indexes.
- JSONB for raw CSV payload.
- Good support for analytical queries.
- Partial indexes for active assignments/open shifts.

---

## ADR-003: pgx + sqlc instead of ORM

Decision:

```text
Use pgx + sqlc.
```

Why:

- Explicit SQL is better for financial and reconciliation logic.
- Less hidden behavior than ORM.
- Easy to optimize queries.
- Type-safe Go code generated from SQL.

Avoid:

```text
GORM for core domain logic.
```

---

## ADR-004: REST + OpenAPI

Decision:

```text
Use REST API and OpenAPI contract.
```

Why:

- CRM actions map well to resources.
- Easy frontend integration.
- Easy testing.
- GraphQL is unnecessary for MVP.

---

## ADR-005: Server-side sessions, not JWT

Decision:

```text
Use login/password + server-side sessions.
```

Why:

- Closed CRM.
- Session revocation is easy.
- Role changes take effect immediately.
- Safer and simpler than JWT for this case.

Cookie:

```text
httpOnly
secure
sameSite
```

DB stores token hash only.

---

## ADR-006: CSV import is synchronous in MVP

Decision:

```text
Process CSV import synchronously in HTTP request for MVP.
```

Why:

- Example file has ~9k rows, manageable.
- Simpler UX and backend.
- Avoids queues/background workers too early.

Future:

- Add import_jobs table and background worker if files grow.

---

## ADR-007: Preserve import history, replace active scope

Decision:

```text
Do not hard-delete old import data on reimport.
Replace only active scope links.
```

Why:

- Need auditability.
- Need dispute debugging.
- Financial data should be traceable.

Implementation:

- `import_batches` historical.
- `import_rows` historical.
- `order_scope_items.is_active` decides what is current.
- Previous active batch becomes `superseded`.

---

## ADR-008: Money as integer

Decision:

```text
Store money as bigint minor units.
```

Why:

- No float rounding bugs.
- Reconciliation must be exact.

Percent/rates:

```text
salary_rate_bps bigint
```

---

## ADR-009: Frontend SPA, no Next.js

Decision:

```text
React + TypeScript + Vite SPA.
```

Why:

- Closed internal CRM.
- SEO not needed.
- SSR not needed.
- Faster and simpler MVP.

---

## ADR-010: shadcn/ui + Tailwind

Decision:

```text
Use shadcn/ui + Tailwind.
```

Why:

- Modern look.
- Components are copyable/customizable.
- Good fit for CRM forms/tables/dialogs.
- Fast UI development.

---

## ADR-011: TanStack Table and Query

Decision:

```text
Use TanStack Query for server state.
Use TanStack Table for tables.
```

Why:

- CRM has many filtered/paginated lists.
- Import/mutation invalidation is important.
- Tables need filters, sorting, pagination, actions.

---

## ADR-012: Auditing as first-class concern

Decision:

```text
Audit all mutations from MVP.
```

Why:

- Financial operations.
- Teamlead needs history.
- Requisite assignment history is important.
- Reimport and mismatch acceptance must be traceable.

---

## ADR-013: No superadmin panel in MVP

Decision:

```text
Do not implement separate admin panel in MVP.
```

Why:

- Product flow starts with teamlead/trader roles.
- Superadmin/multitenant management can be added later.

---

## ADR-014: Reconciliation append-only runs

Decision:

```text
Keep reconciliation runs historical; latest run is used for current UI.
```

Why:

- Reimport and corrections should be traceable.
- It is useful to see how mismatch changed over time.

Potential optimization later:

- Store current reconciliation status on shift/period for quick access.
