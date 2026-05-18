.PHONY: test bench lint

test:
	go test ./... -v -race

bench:
	go test ./pkg/assignment/... -bench=. -benchmem -benchtime=5s

lint:
	go vet ./...