#!/usr/bin/env bash
set -e
DB_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/resto?sslmode=disable}"
echo "Running migrations against $DB_URL"
for f in $(ls db/*.sql | sort); do
  echo "Applying $f"
  psql "$DB_URL" -f "$f"
done
echo "Migrations applied."
