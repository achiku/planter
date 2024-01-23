.PHONY: test
test:
	docker-compose up -d
	DB_PORT=15432 go test -v ./...

.PHONY: clean
clean:
	docker-compose down -v
