# Frontend Design System

## 1. Visual direction

Style: modern, clean, compact financial dashboard.

Mood:

- trustworthy;
- operational;
- clear;
- calm by default;
- visually loud only for money mismatches and blocking states.

Reference mockup:

```text
docs/assets/ui-mockup.png
```

---

## 2. UI stack

```text
React
TypeScript
Vite
Tailwind CSS
shadcn/ui
TanStack Router
TanStack Query
TanStack Table
React Hook Form
Zod
Recharts
```

---

## 3. Layout

Desktop-first CRM.

Breakpoints:

- MVP can be optimized for desktop/tablet.
- Mobile responsive is nice but not primary.

App layout:

```text
Sidebar width: 240px
Topbar height: 56-64px
Main content max width: fluid
Content padding: 24px
Card gap: 16px
```

Use:

- sticky sidebar;
- sticky table headers where useful;
- modals/drawers for forms.

---

## 4. Color system

Use neutral base + one strong accent.

Suggested:

```text
background: very light slate/gray
surface: white
border: light gray
text primary: near black/slate
text secondary: muted gray
accent: violet/indigo
success: green
warning: amber
error: red
info: blue
```

Do not overuse colors. Tables should be mostly neutral with status badges.

---

## 5. Status badges

Use semantic variants:

```text
success      green
failed       red
warning      amber
info         blue
neutral      gray
processing   purple/indigo
```

Labels:

```text
hand_success -> Успех
corrected -> Исправлен
failed/auto_decline -> Неуспех
cancelled -> Отменён
unknown -> Неизвестно
matched -> Сошлось
mismatch -> Расхождение
accepted_with_comment -> Подтверждено
open -> Открыта
closed -> Закрыта
closed_with_discrepancy -> С расхождением
```

---

## 6. Typography

Use system font or Inter.

Scale:

```text
Page title: 24px / 28px, semibold
Section title: 18px / 24px, semibold
Card label: 12-13px, medium, muted
Card value: 22-28px, semibold
Table text: 13-14px
Form labels: 13px, medium
Help text: 12px, muted
```

---

## 7. Cards

Dashboard card structure:

```text
label
value
secondary delta/status
optional mini-chart/icon
```

Cards should show:

- amount;
- count;
- conversion;
- diff;
- salary.

For mismatch cards, use stronger border/background.

---

## 8. Tables

Tables are critical.

Features:

- server-side pagination;
- server-side sorting;
- filters;
- search;
- compact rows;
- row actions menu;
- sticky action column optional;
- status badges;
- money alignment right;
- date formatting consistent.

Table toolbar:

- search input left;
- filters middle;
- primary action right.

Empty table state:

- short message;
- relevant action button.

Loading:

- skeleton rows.

---

## 9. Forms

Use drawers for most CRUD forms:

- requisite form;
- trader form;
- payout form.

Use dialogs for focused actions:

- import CSV;
- accept mismatch;
- close shift;
- reset password.

Validation:

- inline field errors;
- form-level error for server failures;
- disable submit while loading.

---

## 10. Import UX components

Import dropzone:

- dashed border;
- file icon;
- text: “Перетащите CSV сюда или выберите файл”;
- accepted format info;
- max size info.

Reimport warning:

```text
Этот импорт заменит текущие активные данные по смене/периоду. История предыдущего импорта сохранится.
```

Import result:

- success icon/state;
- row count;
- order count;
- status breakdown;
- totals;
- reconciliation status.

---

## 11. Mismatch UX

Mismatch is a blocking financial state.

Use:

- red alert;
- diff amount large;
- expected vs actual values side by side;
- issue list if available;
- clear actions:
  - “Исправить данные”;
  - “Подтвердить с комментарием”.

Required comment textarea should be impossible to bypass.

---

## 12. Close shift UX

Close shift dialog uses checklist.

Checklist item states:

- done;
- blocked;
- warning;
- missing.

Button:

- disabled if blocked;
- primary if clean close;
- warning/danger style if closing with discrepancy.

---

## 13. Navigation labels

Teamlead sidebar:

```text
Дашборд
Реквизиты
Сотрудники
Входы (платежи)
Выплаты
Периоды
Аудит
```

Trader sidebar:

```text
Мои реквизиты
Входы
Выплаты
Аналитика
```

---

## 14. Formatting

Money:

```text
1 250 000 ₽
```

Percent:

```text
96.2%
```

Date/time:

```text
31.05.2026 14:32
```

Diff:

```text
+10 000 ₽
-100 000 ₽
```

Use right alignment for monetary table cells.

---

## 15. Accessibility basics

- Buttons have clear labels.
- Status is not color-only; badge text matters.
- Dialogs trap focus.
- Inputs have labels.
- Errors are text-visible.
- Keyboard navigation for forms/tables should not be broken.
