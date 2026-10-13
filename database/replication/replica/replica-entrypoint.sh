#!/bin/bash
set -euo pipefail

: "${PRIMARY_HOST:=postgres-primary}"
: "${REPLICATION_PASSWORD:=replpass}"
: "${PGDATA:=/var/lib/postgresql/18/docker}"

set_primary_conninfo_application_name() {
	local conf="${PGDATA}/postgresql.auto.conf"
	[ -n "${REPLICA_APPLICATION_NAME:-}" ] || return 0
	[ -f "$conf" ] || return 0

	if grep -q "application_name=${REPLICA_APPLICATION_NAME}" "$conf"; then
		return 0
	fi

	sed -i "s/ application_name=[^']*//" "$conf"
	sed -i "/^primary_conninfo = '/s/'$/ application_name=${REPLICA_APPLICATION_NAME}'/" "$conf"
}

# on first start only (if PG_VERSION is not present)
if [ ! -s "${PGDATA}/PG_VERSION" ]; then
	echo "Waiting for primary (${PRIMARY_HOST})..."
	until pg_isready -h "$PRIMARY_HOST" -p 5432 -U "$POSTGRES_USER" -d "$POSTGRES_DB"; do
		sleep 2
	done

	echo "Cloning data directory from ${PRIMARY_HOST}..."
	mkdir -p /var/lib/postgresql
	chown -R postgres:postgres /var/lib/postgresql
	gosu postgres rm -rf "${PGDATA:?}"
	gosu postgres mkdir -p "$PGDATA"
	gosu postgres chmod 700 "$PGDATA"
	PGPASSWORD="$REPLICATION_PASSWORD" gosu postgres pg_basebackup \
		-h "$PRIMARY_HOST" \
		-p 5432 \
		-U replicator \
		-D "$PGDATA" \
		-Fp -Xs -P -R

	set_primary_conninfo_application_name
fi

set_primary_conninfo_application_name

exec docker-entrypoint.sh postgres \
	-c hot_standby=on \
	-c listen_addresses='*'
