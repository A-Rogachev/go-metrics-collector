.DEFAULT_GOAL := test

.PHONY: test help

help:   ## - Выводит список команд
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

test:  ## Запуск тестов локально
	metricstest -test.v -test.run=^TestIteration1$$ -binary-path=cmd/server/server

server:  ## Запуск сервера
	go run cmd/server/main.go