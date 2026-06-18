# GitHub Actions

## Workflows

- `CI` запускается на pull request и push в `main`.
- `Deploy` автоматически запускается после успешного `CI` на `main` и деплоит `dev`.
- `Deploy` можно запустить вручную для `dev` или `prod` из вкладки Actions.

## Required Secrets

Repository или environment secrets:

```text
ASTRA_SSH_HOST
ASTRA_SSH_USER
ASTRA_SSH_PRIVATE_KEY
```

Repository или environment variable:

```text
ASTRA_SSH_PORT
```

Если `ASTRA_SSH_PORT` не задан, workflow использует `22`.

## Deploy Behavior

Deploy workflow подключается к VPS, обновляет `/srv/astra-crm/source`, собирает backend/frontend images на сервере, меняет только image tags в `.env` целевого окружения, запускает Docker Compose и проверяет:

- backend `/health`;
- backend `/ready`;
- публичный frontend route;
- SPA fallback route;
- публичный API routing через `/api/v1/auth/me`.

Production deploy запускается только вручную.
