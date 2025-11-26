APP_NAME=reviewer
MAIN_PATH=./cmd/app

.PHONY: run build lint test test-cover bench docker-up docker-down logs clean

run:
	go run $(MAIN_PATH)

build:
	go build -o bin/$(APP_NAME) $(MAIN_PATH)

lint:
	golangci-lint run

test:
	go test -v ./...

test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

bench:
	go test -v -bench=BenchmarkDeactivateTeam -run='^$$' ./internal/handler/... -benchtime=20x

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down -v

logs:
	docker-compose logs -f

clean:
	rm -rf bin/