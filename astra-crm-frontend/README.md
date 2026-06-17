# Astra CRM Frontend

React + TypeScript + Vite SPA для операционной CRM P2P-команд.

## Dev commands

```bash
npm install
npm run dev
npm test
npm run build
```

## Stack

- Vite, React, TypeScript
- Tailwind CSS, shadcn/ui-style primitives
- TanStack Router, Query, Table
- React Hook Form, Zod
- Recharts
- Vitest, React Testing Library

## Structure

```text
src/app          app composition: router, providers, app-level clients
src/pages        route pages grouped by role and product area
src/widgets      large reusable page sections composed from features/entities
src/features     user scenarios: auth, CSV import, reconciliation, period filter
src/entities     domain UI/model building blocks: order, requisite, status, user
src/shared       cross-cutting API, model types, libs, primitive UI
src/test         test harness setup
```

Current route package layout:

```text
src/pages/teamlead
  accounting-periods/
  audit/
  dashboard/
  employees/
  reports/
  requisites/
  transactions/

src/pages/trader
  analytics/
  payouts/
  reports/
  requisites/
  transactions/
```

Inside a route package, keep `page.tsx` as the coordinator for queries, state and page composition. Split local UI by purpose (`tabs.tsx`, `forms.tsx`, `report.tsx`, `planning.tsx`, etc.) instead of growing one page file. If a component is reused between teamlead and trader flows, move it to `shared`, `entities`, `features` or `widgets` instead of copying it between route packages.

Feature-Sliced Design rule of thumb:

- `shared` must not import from `entities`, `features`, `widgets` or `pages`;
- `entities` may use `shared`;
- `features` may use `entities` and `shared`;
- `widgets` may compose `features`, `entities` and `shared`;
- `pages` compose widgets/features/entities for concrete routes.
