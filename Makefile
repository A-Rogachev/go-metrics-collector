.DEFAULT_GOAL := test

.PHONY: test help

help:   ## - Выводит список команд
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

test1:  ## Запуск тестов для итерации 1
	metricstest -test.v -test.run=^TestIteration1$$ -binary-path=cmd/server/server

test2:  ## Запуск тестов для итерации 2
	metricstest -test.v -test.run=^TestIteration2[AB]*$$ -source-path=. -agent-binary-path=cmd/agent/agent

server:  ## Запуск сервера
	go run cmd/server/main.go

agent:
	go run cmd/agent/main.go
