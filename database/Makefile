IMAGE := postgres:18
CONTAINER := socialnet
PGUSER := someuser
PGPASSWORD := somepass
PGDATABASE := socialnet
PGPORT := 5435

.PHONY: run-db stop-db rm-db connect replication-up replication-down replication-ps replication-status

run-db:
	docker run --name $(CONTAINER) -d \
		-e POSTGRES_USER=$(PGUSER) \
		-e POSTGRES_PASSWORD=$(PGPASSWORD) \
		-e POSTGRES_DB=$(PGDATABASE) \
		-v $(PWD)/migrations/init:/docker-entrypoint-initdb.d \
		-p $(PGPORT):5432 \
		$(IMAGE)

stop-db:
	docker stop $(CONTAINER)

rm-db:
	docker rm -f $(CONTAINER)

connect:
	@PGPASSWORD=$(PGPASSWORD) psql -h localhost -p $(PGPORT) -U $(PGUSER) -d $(PGDATABASE)

replication-up:
	docker compose up -d

replication-down:
	docker compose down

replication-ps:
	docker compose ps

replication-status:
	@PGPASSWORD=$(PGPASSWORD) psql -h localhost -p $(PGPORT) -U $(PGUSER) -d $(PGDATABASE) \
		-c "SELECT application_name, client_addr, state, sync_state FROM pg_stat_replication;"
