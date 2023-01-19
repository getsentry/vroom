.PHONY: build run test issuedetection downloader

build:
	./scripts/build.sh

issuedetection:
	go build -o . -ldflags="-s -w" ./cmd/issuedetection

downloader:
	go build -o . -ldflags="-s -w" ./cmd/downloader

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

python-stdlib: scripts/make_python_stdlib.py
	python scripts/make_python_stdlib.py
