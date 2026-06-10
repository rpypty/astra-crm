# Agent Working Rules

Этот файл задает правила выполнения задач из roadmap. Он предназначен для Codex-агента.

## Обязательный контекст

Перед серьезной backend-задачей читать:

```text
AGENTS.md
docs/product-context.md
docs/domain-model.md
docs/database-design.md
docs/csv-imports.md
docs/reconciliation-rules.md
docs/security-and-audit.md
docs/testing-strategy.md
docs/architecture-decisions.md
```

Перед серьезной frontend-задачей читать:

```text
AGENTS.md
docs/product-context.md
docs/ux-flows.md
docs/ui-pages-and-dialogs.md
docs/frontend-design-system.md
docs/api-outline.md
docs/security-and-audit.md
```

Если задача узкая, можно читать только указанные в ней документы, но нельзя игнорировать `AGENTS.md`.

## Границы папок

Backend:

```text
astra-crm-backend
```

Frontend:

```text
astra-crm-frontend
```

Не смешивать backend и frontend код в одной папке. Общие требования не копировать без необходимости, использовать `docs` как источник истины.

## Нельзя нарушать

- Деньги хранить как integer/minor units, не float.
- `salary_rate_bps` хранить в basis points.
- `innerId` является главным внешним ключом CSV.
- CSV separator: `|`.
- `hand_success` и `corrected` считаются успешным inbound.
- Дубликаты `innerId` внутри одного CSV отклоняются.
- Переимпорт заменяет только active scope set, исторические import batches/rows сохраняются.
- У реквизита только одно активное назначение.
- У трейдера только одна открытая смена.
- Смена создается автоматически при взятии первого назначенного реквизита в работу.
- Обороты cumulative, сверка берет latest entry per shift requisite.
- Outbound reconciliation сравнивает CSV payout amount с суммой intermediate transfers.
- Смена не закрывается без inbound/outbound import и matched/accepted reconciliation.
- Любое accepted mismatch требует непустой комментарий.
- Закрытая смена не переоткрывается.
- Все мутации аудитятся.
- Пароли, password hashes, session tokens и token hashes не логируются и не пишутся в audit payload.

## Формат задачи для агента

Каждая задача в roadmap описана крупно. При выполнении агент должен развернуть ее в локальный план:

```text
1. Inspect current repo state.
2. Read relevant docs.
3. Identify existing patterns.
4. Implement focused changes.
5. Add tests.
6. Run verification.
7. Report outcome.
```

## Когда документировать ambiguity

Если решение не задано в docs и влияет на продуктовые правила, не менять поведение молча.

Допустимые действия:

- выбрать safest MVP-compatible option;
- добавить заметку в `docs/open-questions.md` или task notes;
- явно указать assumption в финальном сообщении.

## API стиль

- REST under `/api/v1`.
- Explicit typed responses.
- User-safe errors:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Некоторые поля заполнены неверно",
    "fields": {}
  }
}
```

- Backend validation обязательна, даже если frontend уже валидирует.
- Все запросы доменных данных фильтровать по `team_id` из session context.

## Тестовый минимум

Backend:

- unit tests for pure domain logic;
- parser tests for CSV;
- service tests for invariants;
- integration tests with PostgreSQL for transactions/constraints when needed;
- API/RBAC tests for role boundaries.

Frontend:

- form validation tests for critical forms;
- component tests for blockers and mismatch dialogs;
- build verification;
- Playwright/e2e later for end-to-end flows.

## Команды проверки без внутренней обвязки

В документации и README писать обычные команды:

```bash
go test ./...
npm run build
npm test
docker compose up
```

