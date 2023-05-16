POSTGRES_HOST?=localhost
POSTGRESQL_URL=postgres://postgres:postgres@$(POSTGRES_HOST):5432/postgres?sslmode=disable
MIGRATION_DIR="db/migrations"

default: help

.PHONY: generate
## generate: generates the grpc go sdk and openapi spec.
generate:
	docker run --volume "$(PWD):/workspace" --workdir /workspace bufbuild/buf generate proto/v1

.PHONY: vendor
## vendor: vendor dependencies for faster running/compilation.
vendor:
	docker run --volume "$(PWD):/go/src/app" --workdir /go/src/app golang:1.20 go mod vendor

.PHONY: vendor
## build: builds the binaries into the bin dir. 
build:
	mkdir -p bin
	docker run --env CGO_ENABLED=0 --volume "$(PWD):/go/src/app" --workdir /go/src/app golang:1.20 \
		go build -buildvcs=false -o ./bin/server ./cmd/server && \
		go build -buildvcs=false -cover -o ./bin/servercvg ./cmd/server && \
		go build -buildvcs=false -o ./bin/waitfor ./cmd/waitfor && \
		go build -buildvcs=false -cover -o ./bin/waitforcvg ./cmd/waitfor && \
		go build -buildvcs=false -o ./bin/migrate ./cmd/migrate && \
		go build -buildvcs=false -cover -o ./bin/migratecvg ./cmd/migrate && \
		go build -buildvcs=false -o ./bin/pubsubsetup ./cmd/pubsubsetup && \
		go build -buildvcs=false -cover -o ./bin/pubsubsetupcvg ./cmd/pubsubsetup && \
		go build -buildvcs=false -o ./bin/worker ./cmd/worker && \
		go build -buildvcs=false -cover -o ./bin/workercvg ./cmd/worker

.PHONY: migrate_up
## migrate_up: initialises and run the migrations on the database.
migrate_up:
	docker run -v "$(PWD)/$(MIGRATION_DIR):/migrations" --network host migrate/migrate -path=/migrations/ -database $(POSTGRESQL_URL) up
	
.PHONY: migrate_down
## migrate_down: tears down the database migrations.
migrate_down:
	docker run -v "$(PWD)/$(MIGRATION_DIR):/migrations" --network host migrate/migrate -path=/migrations/ -database $(POSTGRESQL_URL) down -all

.PHONY: dependencies_down
## dependencies_down: tears down the containerised environment necessary to run component-test against.
dependencies_down:
	docker-compose down 
	
.PHONY: dependencies_up
## dependencies_up: starts the containerised environment necessary to run component-tests against.
dependencies_up: dependencies_down
	docker-compose -f docker-compose.yml -f docker-compose.local.yml up test

.PHONY: component_tests
## component_tests: runs component tests against a locally running server and worker. user must run dependencies_up and start the process themselves. [needs Go 1.20]
component_tests:
	go test -v -tags=component ./tests/component

.PHONY: ci
## ci: starts the containerazed environment and run tests against it inside the `test` container.
ci: dependencies_down
	docker-compose up --abort-on-container-exit test 

.PHONY: help
## help: prints this help message.
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'