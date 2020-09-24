all: docker

docker:
	docker build -t samourai-soroban .

docker-static:
	docker build -t samourai-soroban-static . -f Dockerfile.static

compose-build:
	docker-compose build

up: compose-build
	docker-compose up -d

down:
	docker-compose down

test:
	docker run -p 6379:6379 --name=redis_test -d redis:5-alpine
	go test -v ./... -count=1 -run=Test
	docker stop redis_test && docker rm redis_test

.PHONY: docker docker-static compose-build up down test
