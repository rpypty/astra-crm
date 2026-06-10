# astra-crm-backend

Backend for p2p-crm.

## Requirements

- Go 1.26+
- PostgreSQL for readiness checks and future persistence
- goose CLI for migrations
- sqlc CLI for query generation

## Configuration

Environment variables:

```bash
APP_ENV=development
HTTP_ADDR=:8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/astra_crm?sslmode=disable
SESSION_COOKIE_NAME=p2p_crm_session
SESSION_SECURE=false
```

`DATABASE_URL` is optional only for `development` and `test`. In that mode `/health`
works without a database and `/ready` returns `503`.

## Commands

Run tests:

```bash
go test ./...
```

Run API locally:

```bash
go run ./cmd/api
```

Run migrations:

```bash
goose -dir migrations postgres "$DATABASE_URL" up
```

Generate typed SQL:

```bash
sqlc generate
```

## Endpoints

```text
GET /health
GET /ready
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET /api/v1/auth/me
GET /api/v1/teamlead/traders
POST /api/v1/teamlead/traders
GET /api/v1/teamlead/traders/{traderId}
PATCH /api/v1/teamlead/traders/{traderId}
DELETE /api/v1/teamlead/traders/{traderId}
POST /api/v1/teamlead/traders/{traderId}/reset-password
GET /api/v1/teamlead/requisites
POST /api/v1/teamlead/requisites
GET /api/v1/teamlead/requisites/{requisiteId}
PATCH /api/v1/teamlead/requisites/{requisiteId}
DELETE /api/v1/teamlead/requisites/{requisiteId}
POST /api/v1/teamlead/requisites/{requisiteId}/assign
POST /api/v1/teamlead/requisites/{requisiteId}/unassign
GET /api/v1/teamlead/requisites/{requisiteId}/assignment-history
POST /api/v1/teamlead/inbound/import
POST /api/v1/teamlead/outbound/import
GET /api/v1/trader/shift/current
POST /api/v1/trader/shift/current/close
GET /api/v1/trader/shift/current/checklist
GET /api/v1/trader/shift/current/turnovers
POST /api/v1/trader/shift/current/turnovers
GET /api/v1/trader/requisites
POST /api/v1/trader/requisites/{requisiteId}/take
GET /api/v1/trader/shift-requisites
PATCH /api/v1/trader/shift-requisites/{shiftRequisiteId}
GET /api/v1/trader/shift-requisites/{shiftRequisiteId}/turnovers
GET /api/v1/trader/payouts
POST /api/v1/trader/payouts
GET /api/v1/trader/payouts/{payoutId}
PATCH /api/v1/trader/payouts/{payoutId}
DELETE /api/v1/trader/payouts/{payoutId}
POST /api/v1/trader/payouts/{payoutId}/transfers
DELETE /api/v1/trader/payouts/{payoutId}/transfers/{transferId}
POST /api/v1/trader/inbound/import
GET /api/v1/trader/inbound/reconciliation/latest
POST /api/v1/trader/inbound/reconciliation/{runId}/accept
POST /api/v1/trader/outbound/import
```
