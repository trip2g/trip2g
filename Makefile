build-amd64:
	GOOS=linux GOARCH=amd64 go build -o ./tmp/amd64 -ldflags="-s -w" ./cmd/server

deploy: build-amd64
	cd infra && ansible-playbook site.yml

gqlgen:
	go tool github.com/99designs/gqlgen generate

graphqlgen: gqlgen
	./scripts/waitfor localhost:8081
	sleep 1 # avoid a strange error: connect ECONNREFUSED 127.0.0.1:8081
	npm run graphqlgen

sqlc:
	go tool github.com/sqlc-dev/sqlc/cmd/sqlc generate

db-new:
	go tool github.com/amacneil/dbmate/v2 new $(name)

db-up:
	go tool github.com/amacneil/dbmate/v2 up

db-down:
	go tool github.com/amacneil/dbmate/v2 down
