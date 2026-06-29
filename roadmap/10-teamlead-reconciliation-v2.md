# Teamlead Reconciliation V2

Цель: переработать сверку тимлида в полноценный процесс закрытия периода. Тимлид загружает CSV входов/выходов за выбранный период, система прогоняет пайплайн проверок, показывает расхождения и preview изменений, а после подтверждения асинхронно применяет финальные данные TL CSV к единой сущности транзакций. Изменения должны быть видны и тимлиду, и трейдеру.

## Исходный продуктовый контекст

Владелец проекта недоволен текущей страницей "Сверка" по структуре и UX.

Проблемы текущего экрана:

1. На странице постоянно висит желтый alert. Он не должен отображаться на основном экране сверок.
2. Текущий summary-блок под tabs "Входы/Выходы" непонятен: он не объясняет, что именно сверено, почему показывается такой статус и что с этим делать.
3. История сверок сейчас не выглядит как список полноценных сверок. Нужен список run'ов сверки, в каждый из которых можно провалиться и посмотреть детальную информацию по входам, выходам, расхождениям, комментариям и примененным изменениям.

Желаемая UX-модель:

- тимлид открывает страницу "Сверка" и видит историю полноценных сверок;
- основная информация находится не в alert на верхнем уровне, а на детальной странице конкретной сверки;
- в деталях сверки есть выбор направления:
  - входы;
  - выходы;
- расхождения подсвечиваются явно;
- подтверждение расхождений выполняется одним глобальным действием с комментарием;
- изменения статусов транзакций, смены суммы, смены реквизита или трейдера видны в деталях;
- форма загрузки остается понятной: тимлид загружает CSV выписки входов/выходов;
- при загрузке тимлид выбирает период выгрузки транзакций `date_from/date_to`;
- CSV строки фильтруются по выбранному периоду;
- после подтверждения/закрытия сверки форма загрузки очищается.

Желаемая логика закрытия сверки:

- после анализа тимлид имеет два действия:
  - подтвердить закрытие и обновить все транзакции;
  - отклонить сверку и сохранить run в истории без влияния на аналитику;
- при подтверждении транзакции обновляются до состояния TL CSV и начинают влиять на аналитику;
- при отклонении никакие изменения не применяются к аналитике и транзакциям;
- история должна хранить и подтвержденные, и отклоненные сверки.

Критичное правило по транзакциям:

- транзакции трейдера и транзакции, которые видит/обновляет тимлид, должны быть одной сущностью;
- при применении TL-сверки обновления должны быть видны:
  - тимлиду;
  - трейдеру;
  - в транзакциях трейдера;
  - в отчетах трейдера;
  - в аналитике.

Критичное правило по реквизитам:

- реквизиты должны матчиться по телефону и карте;
- у реквизита всегда есть телефон как основа и номер карты как дополнительный стабильный идентификатор;
- карта один раз привязывается к номеру;
- уникальный ключ реквизита: `bank + phone + card_number`;
- если в транзакциях встречаются телефоны и карты, их нужно маппить в один реквизит, если они относятся к одной связке.

Критичное правило по отчетам трейдера:

- после прогона TL-сверки и подтверждения в отчеты трейдера должен прорастать TL-статус:
  - подтверждено TL;
  - обновлено TL;
  - расхождение с TL;
  - принято TL.

## Итоговые решения после уточнений

Пайплайн анализа:

```text
Проверка оборотов -> Потранзакционная проверка -> Preview изменений
```

Важно: потранзакционная проверка по `innerId` обязательна всегда, даже если агрегированные обороты сошлись. Проверка оборотов не является условием запуска transaction diff.

Confirm/apply:

- confirm не применяет изменения синхронно в HTTP-запросе;
- confirm переводит run в apply queue;
- применение выполняется асинхронно;
- inbound и outbound применяются отдельно;
- каждое направление применяется в отдельной глобальной DB-транзакции;
- на время применения берется DB-lock по `team_id + period + direction`;
- retry должен быть идемпотентным.

Reject:

- reject сохраняет run в истории;
- reject не меняет транзакции;
- reject не меняет аналитику;
- reject не меняет TL-статусы на сменах/транзакциях.

## Текущее состояние, которое нужно заменить

Текущая страница и логика сверки смешивают несколько разных сценариев:

- загрузку TL CSV;
- текущую inbound/outbound сверку;
- историю отдельных run'ов;
- подтвержденные расхождения;
- аналитику, которая не всегда читает финальные TL-данные.

Что нужно изменить:

- убрать постоянный желтый alert с основной страницы сверок;
- заменить историю отдельных inbound/outbound run'ов на историю полноценных TL-сверок за период;
- сделать обязательный выбор периода выгрузки `date_from/date_to`;
- всегда выполнять пайплайн `Проверка оборотов -> Потранзакционная проверка -> Preview изменений`;
- после подтверждения применять TL CSV к единой модели транзакций;
- reject должен сохранять сверку в истории, но не менять транзакции и аналитику.

## Ключевые продуктовые правила

- TL-сверка создается как единый run за период.
- Один run может содержать входы, выходы или оба направления.
- Потранзакционная сверка по `innerId` обязательна всегда, даже если обороты сошлись.
- Confirm не применяет изменения синхронно в HTTP-запросе.
- Apply выполняется асинхронно.
- Inbound и outbound применяются в отдельных глобальных DB-транзакциях.
- На apply ставятся DB-lock'и по `team_id + period + direction`.
- `external_orders` остается единой сущностью транзакции для TL и трейдера.
- TL CSV после confirm обновляет существующие транзакции по `innerId` или создает отсутствующие.
- Reject не меняет `external_orders`, смены, реквизиты, TL-статусы и аналитику.

## Реквизиты

Актуальная доменная модель:

```text
Requisite identity = bank + phone + card_number
```

Карта является частью идентичности реквизита. Она не является только дневным полем смены.

Правила матчинга:

- у реквизита есть стабильные `bank_code`, `phone`, `card_number`;
- для сравнения используются normalized phone/card;
- уникальность внутри команды:

```text
team_id + bank_code + normalized_phone + normalized_card_number
```

- если в CSV есть телефон и карта, матчить по `bank + phone + card`;
- если есть только карта, матчить по `bank + card`, если это однозначно;
- если есть только телефон и найдено несколько карт, это `ambiguous_requisite`;
- если карта указывает на другой телефон/банк, это conflict item, не silent update.

## TL-статусы

В отчетах трейдера нужно показывать, как TL-сверка затронула смену и транзакции.

Enum:

```text
not_checked
confirmed_by_tl
updated_by_tl
tl_discrepancy
tl_accepted
```

Где использовать:

- `external_orders.tl_reconciliation_status`;
- `trader_shifts.tl_reconciliation_status`;
- опционально `shift_requisites.tl_reconciliation_status`, если UI нужен статус на уровне реквизита в смене.

Историю конкретных изменений не хранить в enum-флагах. Для этого нужны reconciliation items с `before_json` и `after_json`.

---

## TL-REC-V2-01. Domain and Schema Foundation

Логическая часть: доменная модель, миграции, базовые enum'ы.

Рабочая папка:

```text
docs
astra-crm-backend
```

Что делаем:

- обновить документацию по `Requisite`: карта является частью base identity;
- обновить документацию по TL-сверке и pipeline;
- добавить/актуализировать в `requisites`:
  - `bank_code`;
  - `phone`;
  - `card_number`;
  - `normalized_phone`;
  - `normalized_card_number`;
- добавить уникальность для active/non-deleted реквизитов:

```text
team_id + bank_code + normalized_phone + normalized_card_number
```

- добавить TL-статусы в нужные таблицы:
  - `external_orders`;
  - `trader_shifts`;
  - опционально `shift_requisites`;
- добавить новую сущность run'а TL-сверки:

```text
teamlead_reconciliations
```

- добавить сущность деталей/изменений:

```text
teamlead_reconciliation_items
```

Минимальные статусы run'а:

```text
draft
analyzing
matched
mismatch
apply_queued
applying
applied
apply_failed
rejected
```

Done:

- схема поддерживает `bank + phone + card_number`;
- есть единая сущность TL-сверки за период;
- есть место для хранения diff/preview/apply items;
- существующие данные мигрируют без потери истории.

---

## TL-REC-V2-02. Backend Analysis Pipeline

Логическая часть: создание сверки, импорт CSV, нормализация, матчинг и анализ.

Рабочая папка:

```text
astra-crm-backend
```

Что делаем:

- добавить backend service для TL reconciliation run;
- создать endpoint создания сверки с периодом и файлами:

```text
POST /api/v1/teamlead/reconciliations
```

- request:
  - `dateFrom`;
  - `dateTo`;
  - inbound CSV, optional;
  - outbound CSV, optional;
- минимум один CSV обязателен;
- период обязателен;
- CSV строки вне периода не участвуют в анализе;
- duplicate `innerId` внутри одного файла отклоняет импорт;
- реализовать pipeline:

```text
1. Нормализация CSV
2. Матчинг trader/requisite
3. Проверка оборотов
4. Обязательная потранзакционная проверка
5. Preview изменений
```

Матчинг:

- `workerName -> trader_profiles.external_worker_name`;
- `bank + normalized_phone + normalized_card_number -> requisite_id`.

Turnover check:

- inbound: TL CSV success/corrected vs CRM turnover по shift requisites за период;
- outbound: TL CSV payout success vs CRM payout transfers/manual payout data;
- historical `worked_discrepancy` должен участвовать в CRM turnover.

Transaction check:

- всегда выполняется после turnover check;
- сравнивает по:

```text
team_id + direction + external_inner_id
```

- фиксирует:
  - missing in CRM;
  - missing in TL;
  - amount changed;
  - status changed;
  - trader changed;
  - requisite changed;
  - date changed;
  - unmatched trader/requisite;
  - ambiguous requisite.

Preview:

- сколько транзакций будет создано;
- сколько обновлено;
- сколько не изменится;
- сколько заблокировано конфликтами;
- какие смены/трейдеры/реки будут затронуты.

Done:

- можно создать TL-сверку за период;
- pipeline всегда проходит turnover и transaction stages;
- diff items содержат `before_json`/`after_json`;
- preview не меняет доменные данные.

---

## TL-REC-V2-03. Async Confirm, Reject and Apply

Логическая часть: решение тимлида, асинхронное применение, консистентность.

Рабочая папка:

```text
astra-crm-backend
```

Что делаем:

- добавить endpoints:

```text
POST /api/v1/teamlead/reconciliations/{id}/confirm
POST /api/v1/teamlead/reconciliations/{id}/reject
```

Confirm:

- проверяет, что анализ завершен;
- требует comment, если есть mismatch/conflict;
- переводит run в `apply_queued`;
- создает async apply job;
- не применяет изменения синхронно;
- пишет audit.

Reject:

- переводит run в `rejected`;
- сохраняет comment;
- не меняет `external_orders`, TL-статусы и аналитику;
- пишет audit.

Async apply:

- MVP: допустим in-process worker с persisted job state в БД;
- inbound и outbound применяются отдельно;
- для каждого direction:

```text
BEGIN;
pg_advisory_xact_lock(hash(team_id, date_from, date_to, direction));
apply direction;
write apply result;
COMMIT;
```

- если inbound applied, а outbound failed, run получает `apply_failed` с direction-level результатом;
- retry должен быть идемпотентным;
- параллельный apply на тот же `team_id + period + direction` должен блокироваться.

Применение:

- upsert `external_orders` по:

```text
team_id + direction + external_inner_id
```

- существующие транзакции обновляются до TL CSV;
- отсутствующие транзакции создаются;
- unchanged транзакции получают `confirmed_by_tl`;
- измененные транзакции получают `updated_by_tl`;
- принятые расхождения получают `tl_accepted`;
- затронутые смены получают актуальный TL-статус;
- все изменения аудируются.

Done:

- confirm быстрый и не блокирует HTTP тяжелой операцией;
- apply консистентен на уровне БД;
- retry безопасен;
- TL CSV становится финальным источником транзакций после успешного apply.

---

## TL-REC-V2-04. Readmodels, API Details and Analytics

Логическая часть: отдача истории/деталей, аналитика после применения, трейдерские отчеты.

Рабочая папка:

```text
astra-crm-backend
docs
```

Что делаем:

- добавить endpoints:

```text
GET /api/v1/teamlead/reconciliations
GET /api/v1/teamlead/reconciliations/{id}
GET /api/v1/teamlead/reconciliations/{id}/items
```

- history response:
  - id;
  - period;
  - status;
  - createdBy;
  - createdAt;
  - inbound summary;
  - outbound summary;
  - mismatch count;
  - apply status;
- details response:
  - pipeline stages;
  - direction summaries;
  - preview;
  - comment;
  - apply result;
- items filters:
  - direction;
  - stage;
  - issue type;
  - severity;
  - trader;
  - requisite;
  - only mismatches;
- обновить teamlead dashboard/list, чтобы примененные TL данные учитывались в аналитике;
- обновить trader transactions/reports, чтобы трейдер видел:
  - обновленные транзакции;
  - TL-статус смены;
  - TL-статус транзакции;
- обновить OpenAPI и frontend generated types/API client.

Done:

- история показывает полноценные сверки, а не отдельные inbound/outbound run'ы;
- детали содержат все данные для UI pipeline;
- примененная TL-сверка влияет на аналитику;
- rejected run не влияет на аналитику.

---

## TL-REC-V2-05. Frontend Reconciliation UX

Логическая часть: новая страница сверок, форма загрузки, детали run'а.

Рабочая папка:

```text
astra-crm-frontend
```

Что делаем:

- перестроить страницу "Сверка";
- убрать постоянный желтый alert;
- главный экран: история полноценных сверок;
- таблица истории:
  - период;
  - дата запуска;
  - автор;
  - статус;
  - входы summary;
  - выходы summary;
  - расхождения;
  - apply status;
  - действия;
- кнопка "Загрузить сверку" открывает dialog/drawer;
- форма загрузки:
  - `dateFrom`;
  - `dateTo`;
  - inbound CSV;
  - outbound CSV;
  - минимум один CSV обязателен;
  - после successful confirm/reject локальное состояние формы очищается;
- детальная страница run'а:
  - stepper:

```text
Проверка оборотов -> Потранзакционная проверка -> Preview изменений -> Применение
```

  - direction switch `Входы / Выходы`;
  - таблица items;
  - фильтры по типу, severity, трейдеру, реквизиту;
  - actions `Подтвердить и применить` / `Отклонить`;
  - state `apply_queued/applying/apply_failed/applied`;
- добавить TL-бейджи в trader reports/transactions:

```text
not_checked -> Не проверено TL
confirmed_by_tl -> Подтверждено TL
updated_by_tl -> Обновлено TL
tl_discrepancy -> Расхождение с TL
tl_accepted -> Принято TL
```

Done:

- основной экран отвечает на вопрос "какие сверки были и в каком они статусе";
- важная информация находится в деталях run'а;
- UX больше не строится вокруг непонятного alert/summary блока;
- трейдер видит TL-статус в отчетах и транзакциях.

---

## TL-REC-V2-06. Regression and Release Safety

Логическая часть: тесты, миграционная безопасность, проверка критического пути.

Рабочие папки:

```text
astra-crm-backend
astra-crm-frontend
```

Backend тесты:

- миграции на fresh/existing DB;
- `bank + phone + card` uniqueness;
- matching phone/card в разных форматах;
- ambiguous/conflict requisite cases;
- turnover matched, transaction diff exists;
- turnover mismatch, transaction diff exists;
- transaction check запускается всегда;
- confirm создает async job;
- reject ничего не меняет в данных;
- apply создает missing transaction;
- apply обновляет existing transaction;
- apply обновляет TL-статусы;
- parallel apply защищен lock'ом;
- retry идемпотентен.

Frontend проверки:

- история сверок;
- создание сверки с периодом;
- детали pipeline;
- confirm/reject flows;
- applying/apply_failed/applied states;
- TL-бейджи в trader reports/transactions.

Done:

- критический путь закрытия TL-сверки покрыт тестами;
- старые current inbound/outbound endpoints не используются новой страницей;
- миграции не ломают существующие historical данные;
- `go test ./...` и frontend build/test проходят в scope изменений.

## Рекомендуемый порядок выполнения

1. `TL-REC-V2-01` - domain/schema foundation.
2. `TL-REC-V2-02` - backend analysis pipeline.
3. `TL-REC-V2-03` - async confirm/reject/apply.
4. `TL-REC-V2-04` - readmodels/API details/analytics.
5. `TL-REC-V2-05` - frontend UX.
6. `TL-REC-V2-06` - regression and release safety.

## Definition of Done для stream

- TL-сверка создается как единый run за период.
- Период обязателен.
- Потранзакционная проверка выполняется всегда.
- UI показывает pipeline проверок.
- Confirm запускает async apply job.
- Inbound и outbound применяются отдельными DB-транзакциями с lock'ами.
- TL CSV обновляет единую транзакцию в `external_orders`.
- Изменения видны и тимлиду, и трейдеру.
- Reject сохраняет run, но не меняет транзакции и аналитику.
- Реквизиты матчатся по `bank + phone + card_number`.
- В отчетах трейдера отображаются TL-статусы.
- История сверок и детальная страница заменяют текущий UX с постоянным alert и разрозненными inbound/outbound run'ами.
