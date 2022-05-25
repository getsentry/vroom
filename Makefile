.PHONY: build run

build:
	./scripts/build.sh

dev: build
	./scripts/run.sh

docker:
	./build/package/docker/build.sh
	./build/package/docker/publish.sh

deploy:
	./deployments/deploy.sh

test:
	go test ./...

format:
	gofmt -l -w -s .
