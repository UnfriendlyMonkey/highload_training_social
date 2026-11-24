ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
DB_DIR := $(ROOT_DIR)/database
BACKEND_DIR := $(ROOT_DIR)/backend
RABBITMQ_COMPOSE := $(ROOT_DIR)/infrastructure/docker-compose.rabbitmq.yml

PGUSER := someuser
PGPASSWORD := somepass
PGDATABASE := socialnet
PGPORT_PRIMARY := 5435

HTTP_ADDR := :3000
DATABASE_URL_MASTER := postgres://$(PGUSER):$(PGPASSWORD)@localhost:$(PGPORT_PRIMARY)/$(PGDATABASE)?sslmode=disable
DATABASE_URL_REPLICA_1 := $(DATABASE_URL_MASTER)
DATABASE_URL_REPLICA_2 := $(DATABASE_URL_MASTER)
REDIS_URL := redis://localhost:6379/0
AMQP_URL ?= amqp://guest:guest@localhost:5672/

export HTTP_ADDR DATABASE_URL_MASTER DATABASE_URL_REPLICA_1 DATABASE_URL_REPLICA_2 REDIS_URL AMQP_URL

.PHONY: help up down db-up db-down wait-db migrate app status check citus-up citus-down citus-db-up citus-wait citus-status rabbitmq-up rabbitmq-wait rabbitmq-down

help:
	@echo "Targets:"
	@echo "  make up      - start Citus and RabbitMQ, apply migrations, run app"
	@echo "  make db-up   - start the Citus cluster"
	@echo "  make migrate - apply numbered SQL migrations to the coordinator"
	@echo "  make app     - run backend against the Citus coordinator"
	@echo "  make down    - stop Citus and RabbitMQ"
	@echo "  make rabbitmq-up/down - start or stop RabbitMQ independently"
	@echo "  make status  - show active Citus workers"
	@echo "  make check   - Citus status + sample API call"

up:
	$(MAKE) citus-up
	$(MAKE) rabbitmq-up
	$(MAKE) rabbitmq-wait
	$(MAKE) app

db-up: citus-db-up

wait-db: citus-wait

migrate:
	@for migration in $(DB_DIR)/migrations/[0-9]*.sql; do \
		echo "Applying $$migration"; \
		PGPASSWORD=$(PGPASSWORD) psql -X -v ON_ERROR_STOP=1 --single-transaction \
			-h localhost -p $(PGPORT_PRIMARY) -U $(PGUSER) -d $(PGDATABASE) \
			-f "$$migration" || exit 1; \
	done

app:
	cd $(BACKEND_DIR) && DATABASE_URL_MASTER=$(DATABASE_URL_MASTER) DATABASE_URL_REPLICA_1=$(DATABASE_URL_MASTER) DATABASE_URL_REPLICA_2=$(DATABASE_URL_MASTER) go run ./cmd/app

down: citus-down rabbitmq-down

status: citus-status

check: status
	@curl -sf "http://localhost:3000/user/search?first_name=test&last_name=test" >/dev/null \
		&& echo "API: ok (http://localhost:3000)" \
		|| echo "API: not running — use 'make app' or 'make up'"

citus-up:
	$(MAKE) citus-db-up
	$(MAKE) citus-wait
	$(MAKE) migrate

citus-db-up:
	@docker rm -f socialnet 2>/dev/null || true
	$(MAKE) -C $(DB_DIR) citus-up

citus-wait:
	@echo "Waiting for Citus coordinator and workers..."
	@ready=0; for i in $$(seq 1 30); do \
		count=$$(PGPASSWORD=$(PGPASSWORD) psql -h localhost -p $(PGPORT_PRIMARY) -U $(PGUSER) -d $(PGDATABASE) -tAc "SELECT count(*) FROM pg_dist_node WHERE groupid <> 0 AND isactive" 2>/dev/null || echo 0); \
		[ "$$count" = "2" ] && ready=1 && break; \
		sleep 2; \
	done; \
	[ $$ready -eq 1 ] || (echo "Citus workers are not ready"; exit 1)

citus-down:
	$(MAKE) -C $(DB_DIR) citus-down

rabbitmq-up:
	docker compose -f $(RABBITMQ_COMPOSE) up -d

rabbitmq-wait:
	docker compose -f $(RABBITMQ_COMPOSE) up -d --wait

rabbitmq-down:
	docker compose -f $(RABBITMQ_COMPOSE) down

citus-status:
	@PGPASSWORD=$(PGPASSWORD) psql -h localhost -p $(PGPORT_PRIMARY) -U $(PGUSER) -d $(PGDATABASE) \
		-c "SELECT * FROM citus_get_active_worker_nodes();"
