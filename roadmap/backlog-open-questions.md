# Backlog and Open Questions

Этот файл содержит вопросы и отложенные задачи. Он не заменяет `docs/open-questions.md`; при изменении продуктового решения нужно обновлять основной docs-файл.

## Product Questions to Resolve

### Money Precision

Текущее safest MVP assumption:

```text
amount_minor BIGINT
3001.0 RUB -> 300100
```

Нужно подтвердить:

- все ли суммы в RUB;
- нужны ли копейки;
- как округлять salary.

### CSV Timezone

CSV даты имеют формат:

```text
DD.MM.YYYY HH:mm:ss
```

Нужно подтвердить timezone внешней админки. До подтверждения использовать явно задокументированную настройку, не timezone локальной машины.

### Shift Boundary vs CSV Range

В примере trader CSV покрывает много дней.

Нужно подтвердить:

- в production trader будет экспортировать CSV строго за смену;
- нужно ли предупреждение о подозрительно широком диапазоне дат.

### Daily Instrument Field

Сейчас docs рекомендуют:

```text
card_number
holder_name
```

Но возможна будущая замена на generic `instrument_value`. В MVP не менять без решения.

### Requisite Matching from CSV

CSV содержит:

```text
requisite
requisitePhone
requisiteId
orderComment
```

MVP assumption:

- trader maps by `workerName`;
- raw requisite fields stored;
- internal `requisite_id` nullable if reliable matching impossible.

### Who Accepts Trader Mismatch

MVP assumption:

- trader can accept own mismatch with required comment.

Possible future:

- teamlead approval for high diff.

### Outbound CSV Statuses

Нужно подтвердить, какие outbound statuses считать expected payout amount. До решения хранить raw statuses и выбирать safest documented rule in reconciliation task.

## Deferred Features

- Superadmin panel.
- Microservices.
- Queues/background import worker.
- WebSocket updates.
- APK/browser extension integrations.
- Bank integrations.
- Advanced notification system.
- Multi-tenant billing.
- Complex RBAC beyond `TEAMLEAD`/`TRADER`.
- Administrative corrections after closed shift.
- Export to CSV/XLSX with formula injection sanitization.

## Technical Backlog

- OpenAPI codegen for frontend client.
- Testcontainers-based integration tests.
- Playwright e2e for critical flows.
- Import jobs/background worker if CSV grows beyond synchronous limits.
- Current reconciliation materialized views or cached read models if dashboards become slow.
- Structured observability dashboards.
- DB backup/restore runbook.

