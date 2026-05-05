GIT_COMMIT := $(shell git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD)
LDFLAGS := -s -w -X main.GitCommit=$(GIT_COMMIT)

test:
	go test ./...

build-amd64: graphqlgen test
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

# bench: 60s load test against http://localhost:8081/ + cpu profile saved to /tmp/trip2g-bench-*
# Usage: make bench              — test homepage
#        make bench URL=/ru      — test specific path
#        make bench RATE=500 DUR=30s  — custom rate and duration
URL  ?= /
RATE ?= 1000
DUR  ?= 60s
PPROF_PORT ?= 8082
BENCH_TS := $(shell date +%Y%m%d_%H%M%S)

bench:
	@echo "==> Starting CPU profile ($(DUR))..."
	@curl -s "http://localhost:$(PPROF_PORT)/debug/pprof/profile?seconds=$(DUR)" \
		-o /tmp/trip2g-cpu-$(BENCH_TS).prof & PROF_PID=$$!; \
	sleep 0.5; \
	echo "==> Running vegeta: $(RATE) req/s for $(DUR) against http://localhost:8081$(URL)"; \
	echo "GET http://localhost:8081$(URL)" \
		| $(HOME)/go/bin/vegeta attack -rate=$(RATE) -duration=$(DUR) \
		| $(HOME)/go/bin/vegeta report -type=text; \
	wait $$PROF_PID; \
	echo "==> CPU profile saved: /tmp/trip2g-cpu-$(BENCH_TS).prof"; \
	echo "==> View: go tool pprof -http=:6060 /tmp/trip2g-cpu-$(BENCH_TS).prof"

bench-max:
	@echo "==> Unlimited rate test ($(DUR)) — finding saturation point..."
	@curl -s "http://localhost:$(PPROF_PORT)/debug/pprof/profile?seconds=$(DUR)" \
		-o /tmp/trip2g-cpu-max-$(BENCH_TS).prof & PROF_PID=$$!; \
	sleep 0.5; \
	echo "GET http://localhost:8081$(URL)" \
		| $(HOME)/go/bin/vegeta attack -rate=0 -workers=50 -duration=$(DUR) \
		| $(HOME)/go/bin/vegeta report -type=text; \
	wait $$PROF_PID; \
	echo "==> CPU profile: /tmp/trip2g-cpu-max-$(BENCH_TS).prof"
