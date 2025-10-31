.PHONY: up migrate create-admin cleanup-sessions run-cleanup-loop

up:
	docker-compose up --build

migrate:
	DATABASE_URL=${DATABASE_URL} ./scripts/run_migrations.sh

create-admin:
	go run scripts/create_admin.go --restaurant ${RESTAURANT_ID} --email ${ADMIN_EMAIL} --password ${ADMIN_PASSWORD} --role ${ADMIN_ROLE}

cleanup-sessions:
	go run scripts/cleanup_refresh_tokens.go --retention 30

run-cleanup-loop:
	docker-compose run --rm cleanup-worker
