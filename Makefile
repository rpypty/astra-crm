.PHONY: restart-backend restart-frontend restart-app

COMPOSE ?= docker compose

restart-backend:
	$(COMPOSE) up -d --no-deps --build --force-recreate backend

restart-frontend:
	$(COMPOSE) up -d --no-deps --build --force-recreate frontend

restart-app:
	$(COMPOSE) up -d --no-deps --build --force-recreate backend frontend
