---
name: astra-crm-deploy
description: Use when deploying Astra CRM to the project VPS, redeploying dev or prod, checking deployment state, or explaining the manual VPS deployment workflow for this repository.
---

# Astra CRM Deploy

## Inputs

Require an environment:

- `dev` -> `dev.paycrm.space`
- `prod` -> `astra.paycrm.space`

Optional input:

- commit/ref/tag to deploy. If omitted, deploy the repository default branch. In this repo that is currently `origin/main`; treat user wording like "master" as the main production branch unless they explicitly name `origin/master`.

If the environment is missing, ask one short clarification question before touching the VPS.

## Access

Connect to the VPS with:

```bash
ssh astra
```

Prefer doing deploy work as `deploy` when possible. If connecting as `root`, avoid changing unrelated server state.

## VPS Paths

Runtime project paths:

```text
/srv/astra-crm/prod
/srv/astra-crm/dev
```

Each environment directory contains:

```text
compose.yml
.env
```

Source checkout for manual VPS builds should live at:

```text
/srv/astra-crm/source
```

If it does not exist, create it and clone:

```bash
sudo mkdir -p /srv/astra-crm
sudo chown -R deploy:deploy /srv/astra-crm
git clone https://github.com/rpypty/astra-crm.git /srv/astra-crm/source
```

## Ports And Routing

Nginx routes public traffic to localhost-only container ports:

```text
prod frontend: 127.0.0.1:3001
prod backend:  127.0.0.1:8081
dev frontend:  127.0.0.1:3002
dev backend:   127.0.0.1:8082
```

Public routes:

```text
https://astra.paycrm.space/       -> prod frontend
https://astra.paycrm.space/api/*  -> prod backend
https://dev.paycrm.space/         -> dev frontend
https://dev.paycrm.space/api/*    -> dev backend
```

## Manual Deploy Workflow

Set variables:

```bash
ENV=dev # or prod
REF=origin/main # or a user-provided commit/ref/tag
```

Resolve environment values:

```bash
if [ "$ENV" = "prod" ]; then
  ENV_DIR=/srv/astra-crm/prod
  DOMAIN=astra.paycrm.space
elif [ "$ENV" = "dev" ]; then
  ENV_DIR=/srv/astra-crm/dev
  DOMAIN=dev.paycrm.space
else
  echo "ENV must be dev or prod" >&2
  exit 1
fi
```

Update source:

```bash
cd /srv/astra-crm/source
git fetch --all --tags --prune
git checkout --detach "$REF"
SHA=$(git rev-parse --short=12 HEAD)
```

Build images on the VPS:

```bash
BACKEND_IMAGE="astra-crm-backend:${ENV}-${SHA}"
FRONTEND_IMAGE="astra-crm-frontend:${ENV}-${SHA}"

docker build -t "$BACKEND_IMAGE" ./astra-crm-backend
docker build \
  -f deploy/Dockerfile.frontend \
  --build-arg VITE_API_BASE_URL=/api/v1 \
  -t "$FRONTEND_IMAGE" \
  .
```

Update the target environment image tags:

```bash
sed -i "s|^BACKEND_IMAGE=.*|BACKEND_IMAGE=${BACKEND_IMAGE}|" "$ENV_DIR/.env"
sed -i "s|^FRONTEND_IMAGE=.*|FRONTEND_IMAGE=${FRONTEND_IMAGE}|" "$ENV_DIR/.env"
```

Deploy with Docker Compose:

```bash
cd "$ENV_DIR"
docker compose --env-file .env -f compose.yml config --quiet
docker compose --env-file .env -f compose.yml up -d --remove-orphans
docker compose --env-file .env -f compose.yml ps
```

## Verification

Check containers:

```bash
cd "$ENV_DIR"
docker compose --env-file .env -f compose.yml ps
docker compose --env-file .env -f compose.yml logs --tail=100 backend frontend migrate
```

Check backend locally on the VPS:

```bash
. "$ENV_DIR/.env"
curl -fsS "http://127.0.0.1:${HOST_BACKEND_PORT}/health"
curl -fsS "http://127.0.0.1:${HOST_BACKEND_PORT}/ready"
```

Check public routing:

```bash
curl -I "https://${DOMAIN}/"
curl -I "https://${DOMAIN}/some/spa/path"
curl -I "https://${DOMAIN}/api/v1/auth/me"
```

Expected:

- frontend paths should not return `502`;
- `/api/v1/auth/me` may return `401` without a session, but it must route to backend and not return `502`;
- if `/ready` fails, inspect PostgreSQL and migrations before declaring deploy successful.

## Rollback

Rollback means restoring previous `BACKEND_IMAGE` and `FRONTEND_IMAGE` values in the target `.env`, then:

```bash
cd "$ENV_DIR"
docker compose --env-file .env -f compose.yml up -d --remove-orphans
docker compose --env-file .env -f compose.yml ps
```

Before changing `.env`, capture previous image values in the task notes or final response.

## Safety Rules

- Never deploy both `dev` and `prod` unless the user explicitly asks.
- Never overwrite `.env` wholesale; update only image tag lines unless the user requests config changes.
- Never expose PostgreSQL ports publicly.
- Do not include real secrets from VPS `.env` in chat, commits, logs, or docs.
- If a command fails, stop and inspect logs before retrying.
