# Integration and Release

Цель milestone: собрать backend и frontend в рабочую систему, подготовить smoke scenarios, Docker Compose и базовое production hardening для VPS.

## INT-001. OpenAPI Contract Alignment

Рабочие папки:

```text
astra-crm-backend
astra-crm-frontend
```

Контекст:

```text
docs/api-outline.md
docs/security-and-audit.md
```

Задача:

- добавить или актуализировать OpenAPI spec;
- синхронизировать frontend types/client;
- убрать расхождения naming/casing;
- зафиксировать common error format.

Done:

- frontend не держит ручные догадки о response shape;
- backend endpoints documented;
- auth cookie behavior documented.

## INT-002. Docker Compose Dev Environment

Контекст:

```text
docs/architecture-decisions.md
docs/database-design.md
```

Задача:

- PostgreSQL container;
- backend container;
- frontend dev/static container as needed;
- env examples;
- migration command documented.

Done:

- fresh clone can run local stack;
- migrations apply;
- health endpoint works from compose network.

## INT-003. Seed Data

Контекст:

```text
docs/product-context.md
docs/domain-model.md
docs/ux-flows.md
```

Задача:

- seed team;
- seed one teamlead;
- seed several traders with `external_worker_name`;
- seed requisites and assignments;
- seed active shift scenario optional.

Done:

- frontend can be manually explored without creating all data from scratch;
- seed passwords documented for local dev only;
- no production secrets committed.

## INT-004. End-to-End Smoke Scenarios

Контекст:

```text
docs/testing-strategy.md
docs/ux-flows.md
docs/reconciliation-rules.md
```

Задача:

- implement smoke tests for critical flows;
- use API tests first, Playwright later.

Critical flows:

```text
login as teamlead
create trader
create requisite
assign requisite
login as trader
take requisite into work
add turnover
create payout
add transfer
import inbound CSV
import outbound CSV
close shift
```

Mismatch flows:

```text
import CSV with mismatch
accept mismatch with comment
close shift with discrepancy
```

Done:

- smoke tests can run locally;
- tests use sanitized CSV fixtures;
- failures point to product rule, not random setup issue.

## INT-005. Security Hardening Pass

Контекст:

```text
docs/security-and-audit.md
```

Задача:

- check cookies;
- CSRF strategy;
- login rate limit;
- CSV upload size limit;
- request logging redaction;
- audit redaction;
- no secrets in repo;
- user-safe DB errors.

Done:

- security checklist documented;
- tests cover sensitive audit/log cases where practical.

## INT-006. Deployment MVP

Контекст:

```text
docs/architecture-decisions.md
```

Задача:

- backend Dockerfile;
- frontend production build;
- Caddy or Nginx config;
- PostgreSQL env;
- migration run strategy;
- basic backup notes.

Done:

- VPS deployment steps documented;
- static SPA served;
- backend behind reverse proxy;
- health/readiness available.

