#!/bin/bash
set -euo pipefail

: "${REPLICATION_PASSWORD:=replpass}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
	SELECT pg_create_physical_replication_slot('replica1', true, false)
	WHERE NOT EXISTS (
		SELECT 1 FROM pg_replication_slots WHERE slot_name = 'replica1'
	);
	SELECT pg_create_physical_replication_slot('replica2', true, false)
	WHERE NOT EXISTS (
		SELECT 1 FROM pg_replication_slots WHERE slot_name = 'replica2'
	);

	DO \$\$
	BEGIN
		IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'replicator') THEN
			CREATE ROLE replicator WITH REPLICATION LOGIN ENCRYPTED PASSWORD '${REPLICATION_PASSWORD}';
		END IF;
	END
	\$\$;
EOSQL

{
	echo "host replication replicator all scram-sha-256"
	echo "host all all all scram-sha-256"
} >> "$PGDATA/pg_hba.conf"
