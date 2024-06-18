.DEFAULT_GOAL := help

#help:	@ List available tasks on this project
help:
	@grep -E '[a-zA-Z\.\-]+:.*?@ .*$$' $(MAKEFILE_LIST)| tr -d '#'  | awk 'BEGIN {FS = ":.*?@ "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


#test: @ Runs unit tests
test:
	go test -race -v -coverprofile=coverage.txt -covermode=atomic

#test.examples: @ Runs examples
test.examples:
	go test -race -v ./examples

#bench: @ Runs performance tests
bench:
	go test -bench=.

#lint: @ Lints source code
lint:
	docker run --rm -v ${CURDIR}:/app -w /app golangci/golangci-lint:v1.49.0 golangci-lint run -v

#scan: @ Scans source code dependencies for vulnerabilities
scan:
	go list -json -deps | docker run --rm -i sonatypecommunity/nancy:v1.0.29 sleuth

