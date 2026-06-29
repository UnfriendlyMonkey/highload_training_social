#!/bin/bash
set -euo pipefail

: "${PRIMARY_HOST:=postgres-primary}"
: "${REPLICATION_PASSWORD:=replpass}"
: "${PGDATA:=/var/lib/postgresql/18/docker}"

# on first start only (if PG_VERSION is not present)
if [ ! -s "${PGDATA}/PG_VERSION" ]; then
	echo "Waiting for primary (${PRIMARY_HOST})..."
	until pg_isready -h "$PRIMARY_HOST" -p 5432 -U "$POSTGRES_USER" -d "$POSTGRES_DB"; do
		sleep 2
	done

	echo "Cloning data directory from ${PRIMARY_HOST}..."
	mkdir -p "$PGDATA"
	rm -rf "${PGDATA:?}"/*
	PGPASSWORD="$REPLICATION_PASSWORD" gosu postgres pg_basebackup \
		-h "$PRIMARY_HOST" \
		-p 5432 \
		-U replicator \
		-D "$PGDATA" \
		-Fp -Xs -P -R
fi

exec docker-entrypoint.sh postgres \
	-c hot_standby=on \
	-c listen_addresses='*'
