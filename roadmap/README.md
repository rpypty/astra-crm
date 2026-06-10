# Roadmap

Этот roadmap описывает разработку p2p-crm как серию крупных инкрементов для Codex-агента.

Цель не в том, чтобы за один заход сгенерировать всю CRM, а в том, чтобы каждый этап оставлял проект в рабочем, проверяемом состоянии.

## Репозитории

Работать нужно в двух отдельных папках:

```text
/Users/ashpak/Pet/astra-crm/astra-crm-backend
/Users/ashpak/Pet/astra-crm/astra-crm-frontend
```

Общий контекст и требования лежат здесь:

```text
/Users/ashpak/Pet/astra-crm/AGENTS.md
/Users/ashpak/Pet/astra-crm/docs
```

## Порядок разработки

1. Backend foundation.
2. Database, auth, RBAC, audit core.
3. Backend core domain: traders, requisites, shifts, payouts.
4. Backend imports and reconciliation.
5. Frontend foundation.
6. Frontend feature flows.
7. Integration, release hardening, deployment.

Backend идет первым, потому что основные риски продукта находятся в доменных инвариантах, CSV-импортах, переимпортах, сверке и аудитах.

Frontend можно начинать после появления auth/API foundation, но не стоит ждать полной готовности backend: shared UI, shell, маршруты и формы можно вести параллельно по OpenAPI/fixtures.

## Файлы roadmap

```text
roadmap/00-agent-working-rules.md
roadmap/01-backend-foundation.md
roadmap/02-backend-data-auth-audit.md
roadmap/03-backend-core-domain.md
roadmap/04-backend-imports-reconciliation.md
roadmap/05-frontend-foundation.md
roadmap/06-frontend-flows.md
roadmap/07-integration-release.md
roadmap/backlog-open-questions.md
```

## Как брать задачи

Для каждой задачи агент должен:

1. Прочитать `AGENTS.md` и указанные в задаче документы.
2. Работать только в указанной папке backend или frontend.
3. Дать короткий план.
4. Сделать малый связный набор изменений.
5. Добавить или обновить тесты.
6. Проверить сборку/тесты.
7. В финале кратко перечислить изменения, проверки и оставшиеся риски.

## Definition of done для milestone

Milestone считается готовым, если:

- код компилируется;
- миграции или API не противоречат `docs/domain-model.md` и `docs/database-design.md`;
- ключевые доменные правила покрыты тестами;
- ошибки пользовательские и безопасные;
- аудит пишется для мутаций;
- frontend имеет loading/empty/error states для реализованных экранов;
- destructive actions требуют подтверждения;
- mismatches визуально очевидны.

