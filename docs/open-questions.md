# Open Questions / Risks

These are not blockers for MVP scaffold, but should be clarified before detailed implementation or production release.

## 1. Money precision

CSV amounts look like `3001.0`.

Question:

```text
Store as rubles bigint or minor units bigint?
```

Recommendation:

```text
amount_minor bigint, parse decimals exactly.
```

## 2. External admin timezone

CSV dates are in format:

```text
DD.MM.YYYY HH:mm:ss
```

Question:

```text
Which timezone does the external admin use?
```

Need consistent parsing for period/shift boundaries.

## 3. Shift boundary vs CSV range

Observed trader CSV example covers many days, not necessarily one shift.

Question:

```text
Will traders export CSV strictly for one shift in production?
```

UX recommendation:

- On import, show detected date range.
- Warn if range is suspiciously broad.

## 4. Card number vs phone for daily details

Original requirement said trader fills:

```text
номер карты
ФИО
```

Later wording mentioned:

```text
телефон, ФИО
```

Question:

```text
Should daily instrument field be specific card_number or generic instrument_value?
```

Recommendation:

```text
For MVP keep card_number + holder_name, but UI can label dynamically by method later.
```

## 5. Requisite matching from CSV

CSV has:

```text
requisite
requisitePhone
requisiteId in trader CSV
orderComment may contain card/phone/FIO
```

Question:

```text
How exactly should external CSV requisite map to internal Requisite?
```

MVP recommendation:

- Store raw fields.
- Map trader by `workerName` first.
- Map requisite by phone/card where reliable; otherwise leave `requisite_id` nullable.

## 6. Can teamlead edit closed shift data?

Current decision:

```text
Closed shift cannot be reopened.
```

Question:

```text
Should teamlead be able to add administrative correction after close?
```

If yes, model as separate correction/audit entity, not direct shift reopen.

## 7. Who can accept trader mismatch?

Current assumption:

```text
Trader can accept own mismatch with required comment.
```

Question:

```text
Should teamlead approval be required for mismatch above a threshold?
```

Future feature possible.

## 8. Outbound CSV statuses

User says payout CSV has same structure.

Question:

```text
Are outbound success statuses same as inbound statuses?
```

MVP:

- Store statuses raw.
- For outbound expected amount, sum all relevant payout CSV rows unless product defines status filtering.

## 9. Duplicate innerId inside one CSV

Recommended MVP:

```text
Reject file.
```

Question:

```text
Should last row win instead?
```

Given financial domain, rejection is safer.

## 10. Data sensitivity

Card numbers and holder names are sensitive.

Question:

```text
Should full card number be stored and shown to teamlead?
```

MVP recommendation:

- Store if operationally required.
- Mask in audit and logs.
- Consider masking in tables.

## 11. Salary rounding

Formula:

```text
salary = success_turnover * salary_rate_bps / 10000
```

Question:

```text
How to round salary?
```

MVP recommendation:

- Integer division/floor if using minor units.
- Document display rounding.

## 12. Login uniqueness across teams

Current auth API accepts only:

```text
login
password
```

Question:

```text
Can different teams have users with the same login?
```

MVP assumption:

- One team is expected initially, so lookup by `login` is enough.
- Before true multi-team usage, either make `login` globally unique or add a team/workspace identifier to the login flow.
