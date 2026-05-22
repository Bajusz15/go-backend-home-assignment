.PHONY: build test integration smoke lint up down seed swagger clean

build:
	go build ./...

test:
	go test ./... -race

lint:
	go vet ./...

integration: up-db
	go test -tags=integration ./tests/integration/... -race -v

smoke: up seed
	./scripts/smoke.sh

swagger:
	swag init -g cmd/api/main.go -o docs

up:
	docker compose up --build -d
	@echo "Waiting for API ..."
	@for i in $$(seq 1 30); do curl -sf http://localhost:8080/health > /dev/null 2>&1 && break || sleep 1; done

up-db:
	docker compose up -d db
	@echo "Waiting for Postgres ..."
	@for i in $$(seq 1 15); do docker compose exec db pg_isready -U postgres > /dev/null 2>&1 && break || sleep 1; done

seed:
	docker compose exec -T api /seed

down:
	docker compose down -v

ci: lint test integration smoke
	@echo "All checks passed."

clean: down
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml
