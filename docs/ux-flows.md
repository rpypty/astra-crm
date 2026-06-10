# UX Flows

## 1. UX principles

The product handles money and reconciliation, so UX must be clear, explicit, and safe.

Key principles:

- Always show current scope: shift, period, direction, trader.
- Always show whether user is working with active data or historical import.
- Mismatch states must be visually loud.
- Any destructive or irreversible action requires confirmation.
- Reimport must warn that active data for scope will be replaced.
- Closing shift should be a checklist, not a blind button.
- Tables should be compact, searchable, filterable, and fast.
- Forms should be short and focused.
- Most side edits should happen in drawers/dialogs without leaving context.

---

# 2. Login flow

Screen: Login.

Fields:

- login;
- password;
- remember me optional.

States:

- default;
- loading;
- invalid credentials;
- user disabled;
- server unavailable.

After login:

```text
TEAMLEAD -> /teamlead/dashboard
TRADER   -> /trader/requisites or /trader/shift/current
```

---

# 3. Teamlead navigation

Main sidebar:

```text
Дашборд
Реквизиты
Сотрудники
Входы (платежи)
Выплаты
Периоды
Аудит
```

Global header:

- team name;
- date range picker;
- notifications icon optional;
- current user menu.

---

# 4. Teamlead — requisites list

Goal: manage requisites and assignments.

Page sections:

1. Header:
   - title “Реквизиты”;
   - primary button “Добавить реквизит”.
2. Filters:
   - search by phone;
   - method;
   - status;
   - assigned trader;
   - proxy optional.
3. Table:
   - phone;
   - method;
   - proxy;
   - assigned trader;
   - status;
   - updated at;
   - actions.
4. Row actions:
   - edit;
   - assign/reassign;
   - history;
   - archive/delete.

Empty state:

```text
Реквизитов пока нет. Добавьте первый реквизит, чтобы назначить его трейдеру.
```

---

# 5. Teamlead — requisite form dialog/drawer

Fields:

- phone;
- method dropdown: SBP, C2C, future methods;
- proxy;
- assigned trader dropdown.

Actions:

- save;
- cancel.

Validation:

- phone required;
- method required;
- if assigning trader, trader must be active;
- proxy format can be loose in MVP.

History section:

- assignment date/time;
- previous trader;
- new trader;
- changed by;
- comment.

UX recommendation:

- Use right drawer for create/edit.
- Keep history as tab inside drawer or separate “История изменений” panel.

---

# 6. Teamlead — employees/traders list

Page sections:

1. Header:
   - title “Сотрудники”;
   - button “Добавить трейдера”.
2. Filters:
   - search by login/workerName;
   - status;
   - salary rate range optional.
3. Table:
   - login;
   - external worker name;
   - salary percent;
   - assigned requisites count;
   - active shift status;
   - status;
   - actions.

Actions:

- edit;
- reset password;
- view assigned requisites;
- disable/archive.

---

# 7. Teamlead — trader form

Fields:

- login;
- password for create;
- reset password action for edit;
- salary percent;
- external worker name;
- assigned requisites multi/list.

Password UX:

- On create: allow manual password input or generated temporary password.
- On edit: do not show password. Show “Сбросить пароль”.
- After reset: show generated password once, with copy button.

---

# 8. Teamlead — inbound orders page

Tabs/sections:

1. Dashboard.
2. Orders list.
3. Import history/reconciliation.

Dashboard cards:

- successful turnover + count;
- failed turnover + count;
- conversion;
- total orders;
- corrected count optional;
- unknown statuses warning if present.

Chart:

- success/fail dynamics over date range.

Orders table:

- created time;
- trader;
- workerName;
- requisite;
- method;
- bank/methodName;
- amount;
- status;
- external id/innerId;
- closed time;
- actions/view details.

Import button:

- “Импорт CSV”.

---

# 9. Teamlead — period import flow

Steps:

```text
1. Teamlead selects accounting period.
2. Opens inbound import modal.
3. Uploads CSV.
4. System warns if this scope already has active import.
5. System parses and applies import.
6. System runs reconciliation.
7. Result dialog shows totals and mismatches.
```

If matched:

- green result;
- show counts/amounts;
- button “Закрыть”.

If mismatch:

- red result;
- show diff;
- show mismatch categories;
- link to detailed reconciliation items;
- require comment if accepting.

---

# 10. Teamlead — outbound page

Similar to inbound:

- dashboard;
- list;
- CSV import.

MVP difference:

- teamlead outbound CSV import has no advanced reconciliation triggers yet.
- It should still create import batch, rows, external orders, active scope items.

---

# 11. Trader navigation

Main sidebar:

```text
Мои реквизиты
Входы
Выплаты
Аналитика / Смена
```

Top shift banner:

- if no shift: “Смена ещё не открыта. Возьмите реквизит в работу.”
- if open: “Смена открыта с HH:mm.”
- if mismatch: red alert with action.
- if ready to close: green check.

---

# 12. Trader — assigned requisites

Page sections:

1. Header:
   - title “Мои реквизиты”.
   - current shift status.
2. Cards/table of assigned requisites:
   - phone;
   - method;
   - proxy;
   - status: available / in work;
   - daily card/holder if filled;
   - latest turnover;
   - action: open/take/update/add turnover.

Take into work form:

- card number;
- holder name.

On first take:

- system creates shift;
- show success toast: “Смена открыта”.

---

# 13. Trader — turnover entry flow

Entry point:

- from requisite card;
- from inbound page “Добавить оборот”.

Fields:

- requisite dropdown;
- cumulative amount;
- comment optional.

Important copy:

```text
Введите текущий накопленный оборот по реквизиту, а не прибавку.
```

After save:

- show latest amount;
- append to history list.

History table:

- date/time;
- amount;
- comment;
- created by.

---

# 14. Trader — inbound page

Sections:

1. Shift summary:
   - successful turnover;
   - order count;
   - failed turnover;
   - conversion;
   - salary estimate.
2. Reconciliation card:
   - expected CSV turnover;
   - actual cumulative turnover;
   - diff;
   - status.
3. Actions:
   - import CSV;
   - add turnover.
4. Orders table.

If mismatch:

- red alert;
- show exact diff;
- show CTA: “Проверить обороты”;
- secondary: “Подтвердить расхождение”.

Accept mismatch dialog:

- required textarea comment;
- warning that shift will close/continue as discrepancy;
- confirm button visually explicit.

---

# 15. Trader — payouts page

Sections:

1. Summary:
   - CSV payout amount;
   - manual transfers amount;
   - diff;
   - unpaid payout count.
2. Manual payout orders table:
   - destination bank;
   - destination requisite;
   - amount;
   - paid amount;
   - remaining;
   - status;
   - actions.
3. Actions:
   - create payout;
   - import payout CSV.

Manual payout form:

- destination bank;
- amount;
- destination requisite.

Payout details drawer:

- original payout amount;
- paid amount;
- remaining;
- status;
- transfer form;
- transfer history.

Transfer form:

- source shift requisite dropdown;
- amount;
- comment optional.

Rules shown in UI:

- cannot add transfer above remaining;
- payout becomes paid when remaining reaches zero;
- shift cannot close while remaining > 0.

---

# 16. Trader — close shift flow

Close shift button opens checklist modal.

Checklist:

```text
Входящий CSV загружен
Входы сверены или подтверждены с комментарием
CSV выплат загружен
Выплаты сверены или подтверждены с комментарием
Все ручные выплаты полностью заполнены переводами
```

If all matched:

- green state;
- button “Закрыть смену”.

If accepted discrepancy exists:

- yellow/red state;
- show comments;
- button “Закрыть смену с расхождением”.

If blocked:

- disabled button;
- show exact reason.

---

# 17. Audit UX

Audit table:

- time;
- actor;
- action;
- entity type;
- entity id/display;
- changed fields;
- comment;
- details action.

Details drawer:

- before JSON;
- after JSON;
- changed fields;
- event metadata.

Sensitive values must be redacted.

---

# 18. User feedback patterns

Success toast:

```text
Реквизит сохранён
CSV импортирован
Сверка совпала
Смена закрыта
```

Warning:

```text
Этот импорт заменит текущие активные данные по смене
```

Error:

```text
Смену нельзя закрыть: выплата #123 заполнена не полностью
```

Mismatch alert:

```text
Обнаружено расхождение оборота: -100 000 ₽
```
