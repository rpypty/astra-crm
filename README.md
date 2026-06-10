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

## Local Dev Stack

1. Скопируй `.env.example` в `.env` при необходимости изменить порты или пароли.
2. Запусти локальный stack:

```bash
docker compose up --build
```

Compose поднимает PostgreSQL, применяет SQL migrations, выполняет seed и запускает backend/frontend.

Локальные URL:

- Frontend: http://localhost:5173
- Backend health: http://localhost:8080/health
- Backend readiness: http://localhost:8080/ready
- API base path: http://localhost:8080/api/v1

Seed users для локальной разработки:

```text
teamlead / demo123
trader_ivan / demo123
trader_anna / demo123
trader_oleg / demo123
```

OpenAPI contract лежит в `docs/openapi.yaml`.

## API Smoke

После запуска compose:

```bash
node scripts/smoke-api.mjs
```

Скрипт проходит критический API path: login, create trader, create requisite, take requisite into work, add turnover, create payout, add transfer, import inbound/outbound CSV, close shift, затем отдельный mismatch flow с accept comment и закрытием `closed_with_discrepancy`.
