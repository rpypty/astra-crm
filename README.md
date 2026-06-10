# p2p-crm Context Pack

Этот пакет предназначен для передачи контекста проекта **p2p-crm** в Codex / IDE / новый чат.

Проект: CRM для тимлидов и P2P-трейдеров.

Главная идея MVP: система помогает трейдерам вести смену, фиксировать реквизиты в работе, cumulative-обороты и ручные выплаты, а тимлидам — управлять трейдерами/реками, импортировать CSV за период и сверять фактические данные с CSV из внешней админки.

## Как использовать

1. Скопировать `AGENTS.md` в корень репозитория.
2. Скопировать папку `docs/` в репозиторий.
3. В Codex давать задачи в стиле:

```text
Прочитай AGENTS.md и docs/*.md.
Сделай задачу <...>.
Сначала дай план, затем внеси изменения.
Соблюдай domain rules и invariants из docs/domain-model.md.
```

## Состав

- `AGENTS.md` — постоянные инструкции для Codex по проекту.
- `docs/product-context.md` — продуктовый контекст, роли, MVP scope.
- `docs/domain-model.md` — доменная модель, агрегаты, поля, связи, инварианты.
- `docs/reconciliation-rules.md` — правила сверки входов/выплат/периода.
- `docs/csv-imports.md` — структура CSV, парсинг, переимпорт, статус-маппинг.
- `docs/database-design.md` — PostgreSQL schema draft, индексы, ограничения.
- `docs/api-outline.md` — REST API outline.
- `docs/ux-flows.md` — UX-потоки для тимлида и трейдера.
- `docs/ui-pages-and-dialogs.md` — список страниц, диалогов и UX-состояний.
- `docs/frontend-design-system.md` — UI kit, дизайн-токены, таблицы, статусы.
- `docs/architecture-decisions.md` — стек и архитектурные решения.
- `docs/security-and-audit.md` — безопасность, авторизация, аудит.
- `docs/testing-strategy.md` — тестовая стратегия.
- `docs/codex-task-prompts.md` — готовые prompts для Codex.
- `docs/open-questions.md` — оставшиеся вопросы/риски.
- `docs/assets/ui-mockup.png` — первый визуальный mockup, сгенерированный в чате.

## Важное

Этот пакет фиксирует решения на момент проектирования MVP. Если решение меняется, обновляй соответствующий `.md`, чтобы Codex не работал по устаревшему контексту.
