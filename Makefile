.PHONY: build
build:
	go build -o conflint ./cmd/main

.PHONY: test
test:
	go test ./...

.PHONY: generate
generate:
	godownloader --repo=mumoshu/conflint > ./install.sh

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 make build

.PHONY: image
image: build-linux
	docker build -t mumoshu/conflint:canary .

.PHONY: run
run: image
	docker run --rm -it -v $(PWD):$(PWD) --workdir $(PWD)/testdata/simple mumoshu/conflint:canary conflint run

.PHONY: test-publish
test-publish:
	CONFLINT_VERSION=0.1.0 goreleaser --snapshot --skip-publish --rm-dist

.PHONY: fmt
fmt:
	go fmt ./...
	goimports -d . && goimports -w .
