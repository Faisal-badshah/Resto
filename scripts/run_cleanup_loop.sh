#!/usr/bin/env sh
if [ -z "$DATABASE_URL" ]; then
  echo "DATABASE_URL not set; exiting"
  exit 1
fi

while true; do
  echo "Running cleanup at $(date -u)"
  /usr/local/bin/cleanup --retention 30 || echo "cleanup failed"
  sleep 86400
done
