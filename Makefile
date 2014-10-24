GOFILES=$(wildcard *.go)

chromeify: $(GOFILES) .deps
	go build -v ./...

.deps: $(GOFILES)
	go get -d -v ./...
	touch .deps

.PHONY: test
test: .deps
	go test -v ./...

.PHONY: serve
serve: chromeify
	./chromeify --addr :8080
