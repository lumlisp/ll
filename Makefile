.PHONY: build test check run clean

build:
	go build -o ll .

test:
	go test ./... -count=1

check: build test
	./test-a-ll

run: build
	./ll $(file)

errors: build
	./test_errors.sh

clean:
	rm -f ll

.DEFAULT_GOAL := build
