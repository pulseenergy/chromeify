GOFILES=$(wildcard *.go) bindata.go

chromeify: $(GOFILES) .deps
	go build -v ./...

$(GOPATH)/bin/go-bindata:
	go get -v github.com/jteeuwen/go-bindata/...

bindata.go: $(GOPATH)/bin/go-bindata data/*
	$(GOPATH)/bin/go-bindata data/

.deps: $(GOFILES)
	go get -d -v ./...
	touch .deps

.PHONY: test
test: .deps
	go test -v ./...

.PHONY: serve
serve: chromeify
	./chromeify --addr :8080
