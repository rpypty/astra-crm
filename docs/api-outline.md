# REST API Outline

Base path example:

```text
/api/v1
```

Authentication via server-side session cookie.

---

## 1. Auth

```text
POST /auth/login
POST /auth/logout
GET  /auth/me
POST /auth/change-password
```

Login request:

```json
{
  "login": "trader_1",
  "password": "secret"
}
```

Me response:

```json
{
  "user": {
    "id": 1,
    "teamId": 1,
    "role": "trader",
    "login": "trader_1",
    "status": "active"
  }
}
```

---

# 2. Teamlead — traders

```text
GET    /teamlead/traders
POST   /teamlead/traders
GET    /teamlead/traders/{traderId}
PATCH  /teamlead/traders/{traderId}
DELETE /teamlead/traders/{traderId}
POST   /teamlead/traders/{traderId}/reset-password
GET    /teamlead/traders/{traderId}/requisites
GET    /teamlead/traders/{traderId}/shifts
```

Create trader:

```json
{
  "login": "trader_ivan",
  "password": "temporary-password",
  "salaryRateBps": 50,
  "externalWorkerName": "Bliss_OP2"
}
```

Patch trader:

```json
{
  "salaryRateBps": 75,
  "status": "active"
}
```

---

# 3. Teamlead — requisites

```text
GET    /teamlead/requisites
POST   /teamlead/requisites
GET    /teamlead/requisites/{requisiteId}
PATCH  /teamlead/requisites/{requisiteId}
DELETE /teamlead/requisites/{requisiteId}
POST   /teamlead/requisites/{requisiteId}/assign
POST   /teamlead/requisites/{requisiteId}/unassign
GET    /teamlead/requisites/{requisiteId}/assignment-history
GET    /teamlead/requisites/{requisiteId}/audit
```

Create requisite:

```json
{
  "phone": "+79991234567",
  "methodType": "sbp",
  "proxy": "192.168.1.1:8080",
  "assignedTraderId": 12
}
```

Assign:

```json
{
  "traderId": 12,
  "comment": "Переназначение на дневную смену"
}
```

---

# 4. Trader — current shift

```text
GET  /trader/shift/current
POST /trader/shift/current/close
GET  /trader/shift/current/checklist
```

Current shift response should include:

```json
{
  "shift": {
    "id": 100,
    "status": "open",
    "startedAt": "...",
    "inboundReconciliationStatus": "matched",
    "outboundReconciliationStatus": "mismatch"
  },
  "checklist": {
    "inboundImported": true,
    "inboundOk": true,
    "outboundImported": true,
    "outboundOk": false,
    "allPayoutsFullyPaid": true,
    "canClose": false
  }
}
```

---

# 5. Trader — requisites

```text
GET  /trader/requisites
GET  /trader/requisites/{requisiteId}
POST /trader/requisites/{requisiteId}/take
PATCH /trader/shift-requisites/{shiftRequisiteId}
GET  /trader/shift-requisites
```

Take requisite into work:

```json
{
  "cardNumber": "1234 5678 9012 3456",
  "holderName": "Иванов Иван Иванович"
}
```

This endpoint:

- verifies the requisite is assigned to trader;
- creates current shift if absent;
- creates `shift_requisite`;
- writes audit.

---

# 6. Trader — turnovers

```text
GET  /trader/shift/current/turnovers
POST /trader/shift/current/turnovers
GET  /trader/shift-requisites/{shiftRequisiteId}/turnovers
```

Create turnover entry:

```json
{
  "shiftRequisiteId": 55,
  "amountMinor": 12500000,
  "comment": "Промежуточный оборот на 15:00"
}
```

---

# 7. Trader — inbound

```text
GET  /trader/inbound/dashboard
GET  /trader/inbound/orders
POST /trader/inbound/import
GET  /trader/inbound/reconciliation/latest
POST /trader/inbound/reconciliation/{runId}/accept
```

Accept mismatch:

```json
{
  "comment": "Расхождение подтверждено: один ордер исправлен вручную во внешней админке"
}
```

Comment is required.

---

# 8. Trader — payouts/outbound

```text
GET    /trader/payouts
POST   /trader/payouts
GET    /trader/payouts/{payoutId}
PATCH  /trader/payouts/{payoutId}
DELETE /trader/payouts/{payoutId}
POST   /trader/payouts/{payoutId}/transfers
DELETE /trader/payouts/{payoutId}/transfers/{transferId}
GET    /trader/outbound/dashboard
GET    /trader/outbound/orders
POST   /trader/outbound/import
GET    /trader/outbound/reconciliation/latest
POST   /trader/outbound/reconciliation/{runId}/accept
```

Create payout:

```json
{
  "destinationBank": "Сбербанк",
  "destinationRequisite": "2200 0000 0000 0000",
  "amountMinor": 1500000
}
```

Add transfer:

```json
{
  "sourceShiftRequisiteId": 77,
  "amountMinor": 500000,
  "comment": "Первая часть выплаты"
}
```

---

# 9. Teamlead — inbound/outbound orders

```text
GET  /teamlead/inbound/dashboard
GET  /teamlead/inbound/orders
POST /teamlead/inbound/import
GET  /teamlead/inbound/reconciliation/latest

GET  /teamlead/outbound/dashboard
GET  /teamlead/outbound/orders
POST /teamlead/outbound/import
```

Filters for lists:

```text
dateFrom
dateTo
traderId
workerName
requisite
methodType
status
amountFrom
amountTo
page
pageSize
sort
```

---

# 10. Teamlead — accounting periods

```text
GET  /teamlead/periods
POST /teamlead/periods
GET  /teamlead/periods/{periodId}
PATCH /teamlead/periods/{periodId}
POST /teamlead/periods/{periodId}/close
GET  /teamlead/periods/{periodId}/reconciliation/inbound
GET  /teamlead/periods/{periodId}/reconciliation/items
```

---

# 11. Audit

```text
GET /audit
GET /audit/entities/{entityType}/{entityId}
```

Filters:

```text
actorId
action
entityType
entityId
dateFrom
dateTo
page
pageSize
```

---

# 12. Common response patterns

List response:

```json
{
  "items": [],
  "page": 1,
  "pageSize": 50,
  "total": 1245
}
```

Error response:

```json
{
  "error": {
    "code": "SHIFT_CANNOT_BE_CLOSED",
    "message": "Смену нельзя закрыть: выплаты не полностью заполнены переводами",
    "details": []
  }
}
```

Validation error:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Некоторые поля заполнены неверно",
    "fields": {
      "comment": "Комментарий обязателен при расхождении"
    }
  }
}
```

---

# 13. Authorization matrix

## TEAMLEAD

Can:

- manage traders;
- manage requisites;
- assign requisites;
- view all team shifts/orders/imports/audit;
- import period CSV;
- view period reconciliation.

Cannot:

- act as trader in a trader shift unless explicit impersonation feature is later added.

## TRADER

Can:

- view assigned requisites;
- take assigned requisites into current shift;
- manage own turnovers;
- manage own manual payouts;
- import own shift CSV;
- accept own shift mismatches with comment;
- close own shift.

Cannot:

- view other traders' data;
- manage requisites assignment;
- manage traders;
- import teamlead period CSV;
- view global audit.
