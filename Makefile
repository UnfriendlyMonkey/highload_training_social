ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
DB_DIR := $(ROOT_DIR)/database
BACKEND_DIR := $(ROOT_DIR)/backend

PGUSER := someuser
PGPASSWORD := somepass
PGDATABASE := socialnet
PGPORT_PRIMARY := 5435
PGPORT_REPLICA1 := 5436
PGPORT_REPLICA2 := 5437

HTTP_ADDR := :3000
DATABASE_URL_MASTER := postgres://$(PGUSER):$(PGPASSWORD)@localhost:$(PGPORT_PRIMARY)/$(PGDATABASE)?sslmode=disable
DATABASE_URL_REPLICA_1 := postgres://$(PGUSER):$(PGPASSWORD)@localhost:$(PGPORT_REPLICA1)/$(PGDATABASE)?sslmode=disable
DATABASE_URL_REPLICA_2 := postgres://$(PGUSER):$(PGPASSWORD)@localhost:$(PGPORT_REPLICA2)/$(PGDATABASE)?sslmode=disable

export HTTP_ADDR DATABASE_URL_MASTER DATABASE_URL_REPLICA_1 DATABASE_URL_REPLICA_2

.PHONY: help up down db-up db-down wait-db migrate app status check

help:
	@echo "Targets:"
	@echo "  make up      - start replicated DB, apply migrations, run app"
	@echo "  make db-up   - start replicated DB only"
	@echo "  make app     - run backend (expects DB on 5435/5436/5437)"
	@echo "  make down    - stop replicated DB"
	@echo "  make status  - show pg_stat_replication"
	@echo "  make check   - replication status + sample API call"

up: db-up wait-db migrate app

db-up:
	@docker rm -f socialnet 2>/dev/null || true
	$(MAKE) -C $(DB_DIR) replication-up

wait-db:
	@echo "Waiting for primary on port $(PGPORT_PRIMARY)..."
	@ready=0; for i in $$(seq 1 30); do \
		PGPASSWORD=$(PGPASSWORD) pg_isready -h localhost -p $(PGPORT_PRIMARY) -U $(PGUSER) -d $(PGDATABASE) >/dev/null 2>&1 && ready=1 && break; \
		sleep 2; \
	done; \
	[ $$ready -eq 1 ] || (echo "primary not ready"; exit 1)
	@echo "Waiting for 2 streaming replicas..."
	@ready=0; for i in $$(seq 1 40); do \
		count=$$(PGPASSWORD=$(PGPASSWORD) psql -h localhost -p $(PGPORT_PRIMARY) -U $(PGUSER) -d $(PGDATABASE) -tAc "SELECT count(*) FROM pg_stat_replication WHERE state='streaming'" 2>/dev/null || echo 0); \
		[ "$$count" = "2" ] && ready=1 && break; \
		sleep 3; \
	done; \
	[ $$ready -eq 1 ] || (echo "replicas not streaming"; exit 1)

migrate:
	@PGPASSWORD=$(PGPASSWORD) psql -h localhost -p $(PGPORT_PRIMARY) -U $(PGUSER) -d $(PGDATABASE) \
		-f $(DB_DIR)/migrations/0003_load_test_events.sql

app:
	cd $(BACKEND_DIR) && go run ./cmd/app

down:
	$(MAKE) -C $(DB_DIR) replication-down

status:
	$(MAKE) -C $(DB_DIR) replication-status

check: status
	@curl -sf "http://localhost:3000/user/search?first_name=test&last_name=test" >/dev/null \
		&& echo "API: ok (http://localhost:3000)" \
		|| echo "API: not running — use 'make app' or 'make up'"
