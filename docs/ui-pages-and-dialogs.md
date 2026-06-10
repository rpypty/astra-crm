# UI Pages and Dialogs

This document lists all MVP screens and dialogs.

Design target: modern, clean, compact financial CRM with clear states and excellent table UX.

---

# 1. Shared pages/components

## Login page

- Fullscreen layout.
- Left/center login card.
- Brand: P2P CRM.
- Login input.
- Password input.
- Submit.
- Error block.

## App shell

- Sidebar navigation.
- Topbar with current role, user menu, date range if relevant.
- Main content area.
- Cards and tables.
- Toasts.

## Common components

- PageHeader.
- DateRangePicker.
- StatusBadge.
- MoneyCell.
- UserCell.
- RequisiteCell.
- ImportButton.
- MismatchAlert.
- ConfirmDialog.
- EntityAuditDrawer.
- EmptyState.
- ErrorState.
- LoadingSkeleton.

---

# 2. Teamlead screens

## 2.1 Teamlead Dashboard

Purpose: quick overview.

Cards:

- inbound successful turnover;
- inbound failed turnover;
- inbound conversion;
- outbound total;
- active traders;
- open discrepancies.

Sections:

- recent imports;
- recent mismatches;
- top traders by turnover.

## 2.2 Requisites List

Route:

```text
/teamlead/requisites
```

Elements:

- title “Реквизиты”;
- button “Добавить реквизит”;
- filters/search;
- table;
- pagination.

Table columns:

- phone;
- method;
- proxy;
- assigned trader;
- status;
- updated at;
- actions.

Actions:

- edit;
- assignment history;
- archive/delete.

## 2.3 Requisite Form Drawer/Dialog

Used for create/edit.

Fields:

- phone;
- method;
- proxy;
- assigned trader.

Secondary area:

- assignment history.

Dialog buttons:

- cancel;
- save.

## 2.4 Requisite Assignment History Dialog

Columns:

- date/time;
- old trader;
- new trader;
- changed by;
- comment.

## 2.5 Employees/Traders List

Route:

```text
/teamlead/traders
```

Elements:

- title “Сотрудники”;
- button “Добавить трейдера”;
- filters/search;
- table.

Columns:

- login;
- external worker name;
- salary percent;
- assigned requisites count;
- current shift status;
- status;
- actions.

## 2.6 Trader Form Drawer/Dialog

Fields:

- login;
- password on create;
- salary percent;
- external worker name;
- assigned requisites list.

Actions:

- save;
- reset password;
- disable/archive.

## 2.7 Reset Password Dialog

States:

- confirm reset;
- generated password shown once;
- copy button.

## 2.8 Inbound Dashboard/List

Route:

```text
/teamlead/inbound
```

Sections:

- analytics cards;
- chart;
- filters;
- order table;
- CSV import button.

Columns:

- time;
- trader;
- workerName;
- requisite;
- method;
- bank/methodName;
- amount;
- status;
- external innerId;
- actions.

## 2.9 Inbound Import Dialog

Fields:

- accounting period selector;
- file picker/drag-drop;
- warning if reimport;
- upload button.

## 2.10 Import Result Dialog

States:

- success;
- mismatch;
- failed.

Shows:

- rows count;
- status breakdown;
- amount totals;
- expected/actual/diff;
- warnings/errors.

## 2.11 Period Reconciliation Details

Route:

```text
/teamlead/periods/{id}/reconciliation
```

Sections:

- summary;
- by trader breakdown;
- issue table.

Issue types:

- missing in trader import;
- extra in trader import;
- amount mismatch;
- status mismatch;
- worker mismatch.

## 2.12 Outbound Dashboard/List

Route:

```text
/teamlead/outbound
```

Similar to inbound, but MVP import has no advanced period reconciliation.

## 2.13 Accounting Periods

Route:

```text
/teamlead/periods
```

List columns:

- date from;
- date to;
- status;
- inbound reconciliation status;
- outbound imported status;
- created by;
- closed at;
- actions.

## 2.14 Audit Log

Route:

```text
/audit
```

Table columns:

- time;
- actor;
- action;
- entity;
- comment;
- details.

Details drawer:

- before;
- after;
- changed fields.

---

# 3. Trader screens

## 3.1 Trader Home / My Requisites

Route:

```text
/trader/requisites
```

Elements:

- current shift banner;
- assigned requisite cards/table;
- latest turnover by requisite;
- status: available / in work;
- actions.

Actions:

- take into work;
- edit daily details;
- add turnover;
- view turnover history.

## 3.2 Take Requisite Dialog

Fields:

- card number;
- holder name.

Copy:

```text
После сохранения реквизит будет взят в работу. Если открытой смены нет — она будет создана автоматически.
```

## 3.3 Add Turnover Dialog

Fields:

- requisite dropdown;
- cumulative amount;
- comment.

Hint:

```text
Введите текущий накопленный оборот по реквизиту, не прибавку.
```

## 3.4 Turnover History Dialog

Columns:

- time;
- amount;
- comment;
- created by.

## 3.5 Trader Inbound Dashboard

Route:

```text
/trader/inbound
```

Cards:

- successful turnover;
- successful count;
- failed amount/count;
- conversion;
- salary estimate;
- shift status.

Reconciliation card:

- CSV amount;
- manual cumulative amount;
- diff;
- status.

Actions:

- import CSV;
- add turnover;
- accept mismatch if needed.

## 3.6 Trader Inbound Orders List

Can be same page under tabs.

Columns:

- time;
- requisite;
- method;
- bank;
- amount;
- status;
- innerId.

## 3.7 Trader Outbound / Payouts

Route:

```text
/trader/payouts
```

Cards:

- CSV payout amount;
- manual transfers amount;
- diff;
- unpaid payout count.

Manual payout table:

- bank;
- destination requisite;
- amount;
- paid;
- remaining;
- status;
- actions.

Actions:

- create payout;
- import payout CSV;
- add transfer.

## 3.8 Create Manual Payout Dialog

Fields:

- destination bank;
- amount;
- destination requisite.

## 3.9 Payout Details Drawer

Shows:

- destination bank/requisite;
- amount;
- paid amount;
- remaining;
- status.

Transfer form:

- source requisite;
- amount;
- comment.

Transfer history table:

- source requisite;
- amount;
- time.

## 3.10 Import CSV Dialog

Used for trader inbound/outbound.

Shows:

- current shift ID/time;
- direction;
- warning on reimport;
- file upload.

## 3.11 Mismatch Confirmation Dialog

Fields:

- required comment.

Copy:

```text
Подтверждение расхождения будет записано в аудит. Смена будет закрыта/помечена как закрытая с расхождением.
```

Buttons:

- cancel;
- confirm.

## 3.12 Close Shift Dialog

Checklist:

- inbound CSV imported;
- inbound reconciliation OK;
- outbound CSV imported;
- outbound reconciliation OK;
- all payout orders fully paid;
- comments added for mismatches.

Button states:

- disabled if blocked;
- “Закрыть смену” if all matched;
- “Закрыть смену с расхождением” if accepted mismatch exists.

---

# 4. Status badge labels

```text
success / hand_success: Успех
corrected: Исправлен
failed / auto_decline: Неуспех
cancelled: Отменён
unknown: Неизвестно
open: Открыта
closed: Закрыта
closed_with_discrepancy: Закрыта с расхождением
mismatch: Расхождение
matched: Сошлось
accepted_with_comment: Подтверждено с комментарием
paid: Выплачено
in_progress: В процессе
```

---

# 5. Critical UX blockers

Never allow closing shift if:

- inbound import missing;
- outbound import missing;
- reconciliation mismatch without comment;
- manual payout not fully paid;
- current shift already closed.

Never allow reimport without showing:

```text
Новый CSV заменит текущие активные данные в этой смене/периоде. История предыдущего импорта сохранится.
```
