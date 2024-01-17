.PHONY: build run test issuedetection downloader python-stdlib gocd

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

python-stdlib:
	python scripts/make_python_stdlib.py

gocd:
	rm -rf ./gocd/generated-pipelines
	mkdir -p ./gocd/generated-pipelines
	cd ./gocd/templates && jb install && jb update

  # Format
	find . -type f \( -name '*.libsonnet' -o -name '*.jsonnet' \) -print0 | xargs -n 1 -0 jsonnetfmt -i
  # Lint
	find . -type f \( -name '*.libsonnet' -o -name '*.jsonnet' \) -print0 | xargs -n 1 -0 jsonnet-lint -J ./gocd/templates/vendor
	# Build
	cd ./gocd/templates && find . -type f \( -name '*.jsonnet' \) -print0 | xargs -n 1 -0 jsonnet --ext-code output-files=true -J vendor -m ../generated-pipelines

  # Convert JSON to yaml
	cd ./gocd/generated-pipelines && find . -type f \( -name '*.yaml' \) -print0 | xargs -n 1 -0 yq -p json -o yaml -i
