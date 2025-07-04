.PHONY: help
help:
	@echo 'Usage'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: run/server
run/server:
	go run ./cmd/server

.PHONY: build
build:
	@echo 'Building cmd/server...'
	go build -o ./bin/gredis ./cmd/server