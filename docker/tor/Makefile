all: docker

docker:
	docker build -t soroban-tor .

docker-static:
	docker build -t soroban-tor-static . -f Dockerfile.static

run:
	docker run soroban-tor

run-static:
	docker run soroban-tor-static

.PHONY: docker run
