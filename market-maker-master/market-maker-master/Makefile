build:
	go build -o bin/market-maker ./cmd/market-maker

test:
	go test ./...

install-deps:
	dep ensure -v

update-deps:
	dep ensure -v -update

run:
	bin/market-maker -cfg etc/market-maker.json

.PHONY: build test install-deps update-deps run
