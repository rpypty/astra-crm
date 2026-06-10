# Frontend Foundation

Рабочая папка:

```text
astra-crm-frontend
```

Цель milestone: создать frontend scaffold и shared UI foundation для плотной операционной CRM.

## FE-001. Initialize Frontend Repository

Контекст:

```text
docs/frontend-design-system.md
docs/ui-pages-and-dialogs.md
docs/ux-flows.md
docs/architecture-decisions.md
```

Задача:

- создать Vite + React + TypeScript app;
- подключить Tailwind;
- подготовить shadcn/ui;
- добавить TanStack Router;
- добавить TanStack Query;
- добавить TanStack Table;
- добавить React Hook Form + Zod;
- добавить Recharts.

Done:

- `npm run build` проходит;
- app стартует локально;
- README содержит dev commands;
- структура готова под role-based pages.

## FE-002. App Shell and Role Navigation

Контекст:

```text
docs/ux-flows.md
docs/ui-pages-and-dialogs.md
docs/frontend-design-system.md
```

Задача:

- реализовать app shell;
- sidebar 240px;
- topbar 56-64px;
- role-aware navigation;
- placeholder pages for teamlead/trader;
- current user menu placeholder.

Teamlead navigation:

```text
Дашборд
Реквизиты
Сотрудники
Входы (платежи)
Выплаты
Периоды
Аудит
```

Trader navigation:

```text
Мои реквизиты
Входы
Выплаты
Аналитика
```

Done:

- route switching works;
- layout follows mockup direction;
- no landing page.

## FE-003. API Client and Query Conventions

Контекст:

```text
docs/api-outline.md
docs/security-and-audit.md
```

Задача:

- создать API client wrapper;
- настроить credentials/cookies;
- common error mapping;
- query keys conventions;
- mutation invalidation conventions.

Done:

- auth endpoints can be wired later without changing UI structure;
- user-safe errors displayed consistently;
- frontend does not trust role from local storage if `/auth/me` exists.

## FE-004. Shared CRM Components

Контекст:

```text
docs/frontend-design-system.md
docs/ui-pages-and-dialogs.md
```

Задача:

- `PageHeader`;
- `StatusBadge`;
- `MoneyCell`;
- `DateTimeCell`;
- `UserCell`;
- `RequisiteCell`;
- `EmptyState`;
- `ErrorState`;
- `LoadingSkeleton`;
- `ConfirmDialog`;
- `EntityAuditDrawer` placeholder.

Rules:

- money right-aligned in tables;
- status must have text, not color-only;
- destructive actions use confirmation;
- compact rows and table toolbar patterns.

Done:

- components used in placeholder pages or story-like demo page;
- visual style matches compact financial CRM.

## FE-005. Table Foundation

Контекст:

```text
docs/frontend-design-system.md
docs/ui-pages-and-dialogs.md
```

Задача:

- create reusable server-side table wrapper using TanStack Table;
- support pagination;
- support sorting;
- support search/filter toolbar;
- support loading/empty/error states;
- support row actions menu.

Done:

- table can power requisites, traders, orders and audit pages;
- no page-specific table hacks in foundation.

## FE-006. Frontend Test Harness

Контекст:

```text
docs/testing-strategy.md
```

Задача:

- add Vitest;
- add React Testing Library;
- add basic component test;
- add build/test scripts.

Done:

- `npm test` passes;
- `npm run build` passes;
- critical future forms can be tested.

