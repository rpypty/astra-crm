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
src/app          app providers, router, query client
src/components   shared UI, CRM cells/states, layout, table foundation
src/lib          API client, query keys, utilities
src/pages        role-based placeholder pages
src/test         test harness setup
```
