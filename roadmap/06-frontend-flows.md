# Frontend Flows

Рабочая папка:

```text
astra-crm-frontend
```

Цель milestone: реализовать основные UI сценарии по ролям. Backend API может быть настоящим или временно замоканным через слой API client, но формы, состояния и блокеры должны соответствовать docs.

## FE-101. Auth and Login Flow

Контекст:

```text
docs/api-outline.md
docs/ux-flows.md
docs/security-and-audit.md
```

Задача:

- login page;
- login/password validation;
- loading/error states;
- `/auth/me` bootstrap;
- redirect by role;
- logout action.

States:

- invalid credentials;
- disabled user;
- server unavailable;
- unauthenticated.

Done:

- role redirects work;
- errors are visible;
- cookie auth assumptions respected.

## FE-102. Teamlead Traders UI

Контекст:

```text
docs/ui-pages-and-dialogs.md
docs/ux-flows.md
docs/frontend-design-system.md
docs/api-outline.md
```

Задача:

- traders list;
- search/status filters;
- create/edit drawer;
- reset password dialog;
- generated password shown once;
- disable/archive confirmation.

Validation:

- login required;
- password required on create;
- salary percent/bps valid;
- external worker name required.

Done:

- loading/empty/error states implemented;
- destructive actions confirmed;
- no password shown on edit.

## FE-103. Teamlead Requisites UI

Контекст:

```text
docs/ui-pages-and-dialogs.md
docs/ux-flows.md
docs/frontend-design-system.md
docs/api-outline.md
```

Задача:

- requisites list;
- filters by phone/method/status/trader;
- create/edit drawer;
- assign/reassign trader;
- assignment history dialog;
- archive/delete confirmation.

Rules:

- card number/holder name must not appear in base requisite form;
- show assigned trader;
- show status badges.

Done:

- table UX is compact and filterable;
- assignment history is visible;
- destructive actions confirmed.

## FE-104. Trader Requisites and Shift UI

Контекст:

```text
docs/ux-flows.md
docs/ui-pages-and-dialogs.md
docs/frontend-design-system.md
docs/api-outline.md
```

Задача:

- my requisites page;
- current shift banner;
- take requisite dialog;
- edit daily details;
- add cumulative turnover dialog;
- turnover history dialog;
- close shift checklist modal.

Important copy:

```text
Введите текущий накопленный оборот по реквизиту, а не прибавку.
```

Done:

- first take flow clearly says shift will be created automatically;
- close button blocked with exact reasons;
- mismatch states can be displayed when backend provides them.

## FE-105. Trader Payouts UI

Контекст:

```text
docs/ux-flows.md
docs/ui-pages-and-dialogs.md
docs/reconciliation-rules.md
```

Задача:

- payouts page;
- summary cards;
- manual payout table;
- create payout dialog;
- payout details drawer;
- add transfer form;
- transfer history;
- unpaid payout blocker display.

Rules:

- cannot add transfer above remaining;
- paid/remaining visible;
- payout becomes paid when remaining is zero;
- shift close blocker visible.

Done:

- validation prevents overpay on frontend;
- backend error still handled safely;
- destructive transfer delete confirmed.

## FE-106. CSV Import and Mismatch Components

Контекст:

```text
docs/csv-imports.md
docs/reconciliation-rules.md
docs/ui-pages-and-dialogs.md
docs/frontend-design-system.md
```

Задача:

- reusable `ImportCsvDialog`;
- reusable `ImportResultDialog`;
- `MismatchAlert`;
- `AcceptMismatchDialog`;
- reimport warning;
- status breakdown;
- expected/actual/diff summary;
- required comment textarea.

Required warning:

```text
Новый CSV заменит текущие активные данные в этой смене/периоде. История предыдущего импорта сохранится.
```

Done:

- mismatch is visually loud;
- accepting mismatch without comment impossible;
- import failure shows row-level duplicate/errors if backend returns details.

## FE-107. Dashboards, Orders, Periods, Audit

Контекст:

```text
docs/ui-pages-and-dialogs.md
docs/ux-flows.md
docs/frontend-design-system.md
docs/api-outline.md
```

Задача:

- teamlead dashboard;
- trader inbound dashboard;
- trader outbound dashboard;
- inbound/outbound order lists;
- accounting periods page;
- period reconciliation details;
- audit log and details drawer.

Done:

- tables support pagination/filter/search where required;
- status badges use Russian labels;
- money/date formatting consistent;
- audit sensitive values are displayed masked if backend sends masked payload.

