APP_NAME=reviewer
MAIN_PATH=./cmd/app

.PHONY: run build docker-up docker-down clean

run:
	go run $(MAIN_PATH)

build:
	go build -o bin/$(APP_NAME) $(MAIN_PATH)

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down -v

logs:
	docker-compose logs -f
	
clean:
	rm -rf bin/