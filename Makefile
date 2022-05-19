.PHONY: build run

build:
	./scripts/build.sh

dev: build
	./scripts/run.sh

deploy:
	./scripts/deploy.sh
