.PHONY: help build run tidy fmt vet test \
        up down restart logs ps psql \
        docker-build clean

BINARY := bin/bot
ENV_FILE := .env

help:
	@echo "Targets:"
	@echo "  build         - compile binary to $(BINARY)"
	@echo "  run           - run bot locally with $(ENV_FILE)"
	@echo "  tidy          - go mod tidy"
	@echo "  fmt           - go fmt ./..."
	@echo "  vet           - go vet ./..."
	@echo "  test          - go test ./..."
	@echo "  up            - docker compose up -d --build"
	@echo "  down          - docker compose down"
	@echo "  restart       - docker compose restart bot"
	@echo "  logs          - tail bot logs"
	@echo "  ps            - docker compose ps"
	@echo "  psql          - open psql shell in postgres container"
	@echo "  docker-build  - build bot image only"
	@echo "  clean         - remove $(BINARY)"

build:
	go build -o $(BINARY) ./cmd/bot

run:
	@test -f $(ENV_FILE) || (echo "$(ENV_FILE) not found; copy from .env.example" && exit 1)
	set -a; . ./$(ENV_FILE); set +a; go run ./cmd/bot

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

up:
	docker compose up -d --build

down:
	docker compose down

restart:
	docker compose restart bot

logs:
	docker compose logs -f bot

ps:
	docker compose ps

psql:
	docker compose exec postgres psql -U $${POSTGRES_USER:-reminder} -d $${POSTGRES_DB:-reminder}

docker-build:
	docker compose build bot

clean:
	rm -rf bin
