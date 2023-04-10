.PHONY: build price-reporter

build: price-reporter

price-reporter:
	go build -o bin/otn-price-reporter .

install-deps:
	dep ensure -v

update-deps:
	dep ensure -update -v
