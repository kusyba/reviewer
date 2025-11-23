.PHONY: build run test clean

build:
	docker-compose build

run:
	docker-compose up

test:
	go test ./...

clean:
	docker-compose down -v
	rm -f main

migrate:
	docker-compose exec postgres psql -U postgres -d pr_reviewer -f /docker-entrypoint-initdb.d/001_init.sql