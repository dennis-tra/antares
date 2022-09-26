default: all

all: clean build

test:
	go test ./...

build:
	go build -o dist/antares cmd/antares/*.go

build-linux:
	GOOS=linux GOARCH=amd64 go build -o dist/antares cmd/antares/*

format:
	gofumpt -w -l .

clean:
	rm -r dist || true

docker:
	docker build . -t dennis-tra/antares:latest

tools:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2
	go install github.com/volatiletech/sqlboiler/v4@v4.13.0
	go install github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-psql@v4.13.0
	go install mvdan.cc/gofumpt@v0.4.0


database:
	docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=password -e POSTGRES_USER=antares -e POSTGRES_DB=antares postgres:14

database-reset: migrate-down migrate-up models

models:
	sqlboiler psql

migrate-up:
	migrate -database 'postgres://antares:password@localhost:5432/antares?sslmode=disable' -path migrations up

migrate-down:
	migrate -database 'postgres://antares:password@localhost:5432/antares?sslmode=disable' -path migrations down

.PHONY: all clean test format tools models migrate-up migrate-down
