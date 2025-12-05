GIT_COMMIT := $(shell git rev-parse --short HEAD)
LDFLAGS := -s -w -X main.GitCommit=$(GIT_COMMIT)

test:
	go test ./...

build-amd64: test
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./tmp/amd64 -ldflags="$(LDFLAGS)" ./cmd/server

build:
	CGO_ENABLED=0 go build -o ./tmp/server -ldflags="$(LDFLAGS)" ./cmd/server

build-docker:
	docker build -t trip2g .
	docker save trip2g | bzip2 > ./tmp/app.tar.bz2

deploy:
	cd infra && ansible-playbook --tags app site.yml

build_and_deploy: build-amd64 deploy

gqlgen:
	go tool github.com/99designs/gqlgen generate

graphqlgen: gqlgen
	./scripts/waitfor localhost:8081
	sleep 1 # avoid a strange error: connect ECONNREFUSED 127.0.0.1:8081
	npm run graphqlgen

sqlc:
	go tool github.com/sqlc-dev/sqlc/cmd/sqlc generate
	./internal/db/fix_write_queries.sh

db-new:
	go tool github.com/amacneil/dbmate/v2 new $(name)

db-up:
	go tool github.com/amacneil/dbmate/v2 up

db-down:
	go tool github.com/amacneil/dbmate/v2 down

lint:
	./internal/db/list_queries.sh
	go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run

docker-deps:
	docker-compose up -d minio

air: docker-deps
	go tool github.com/air-verse/air
