.DEFAULT_GOAL := test

.PHONY: test help

help:   ## - Выводит список команд
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

test1:  ## Запуск тестов для итерации 1
	metricstest -test.v -test.run=^TestIteration1$$ -binary-path=cmd/server/server

test2:  ## Запуск тестов для итерации 2
	metricstest -test.v -test.run=^TestIteration2[AB]*$$ -source-path=. -agent-binary-path=cmd/agent/agent

test3:   ## Запуск тестов для итерации 3
	metricstest -test.v -test.run=^TestIteration3[AB]*$$ -source-path=. -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server

test4:
	SERVER_PORT=8080; \
	ADDRESS="localhost:$${SERVER_PORT}"; \
	TEMP_FILE=$$(./temp.file); \
	metricstest -test.v -test.run=^TestIteration4$$ \
		-agent-binary-path=cmd/agent/agent \
		-binary-path=cmd/server/server \
		-server-port=$$SERVER_PORT \
		-source-path=.


server:  ## Запуск сервера
	go run ./cmd/server/

agent:
	go run ./cmd/agent/

localtest:  ## Запуск юнит и интеграционных тестов
	@echo "\n| ===============> Running agent tests <=============== |\n"
	cd ./cmd/agent && go test . -v
	@echo "\n| ===============> Running server tests <=============== |\n"
	cd ./cmd/server && go test . -v
	@echo "\n| ===============> All TESTS PASSED <=============== |"

build-server:  ## Компиляция сервера
	go build -o ./cmd/server/server ./cmd/server/

build-agent:  ## Компиляция агента
	go build -o ./cmd/agent/agent ./cmd/agent/

statictest:  ## Запуск статического анализатора
	go vet -vettool=$$(pwd)/.tools/statictest ./...
