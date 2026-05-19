#!/bin/bash
set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER}"
DB_NAME="${DB_NAME}"

if [[ -z "$DB_PASSWORD" ]]; then
  echo "Error: DB_PASSWORD environment variable is not set" >&2
  exit 1
fi

echo "Running migrations against $DB_HOST:$DB_PORT/$DB_NAME..."

for file in migrations/*.sql; do
    echo "Applying $file..."
    PGPASSWORD=$DB_PASSWORD psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -f "$file"
done

echo "Migrations complete."