STATS_DIR := stats
STATS_VENV := $(STATS_DIR)/.venv/bin/python

.PHONY: test bench lint stats-proto stats-test stats-run

test:
	go test ./... -v -race

bench:
	go test ./pkg/assignment/... -bench=. -benchmem -benchtime=5s

lint:
	go vet ./...

# Regenerate Python + Go stubs from stats/stats.proto
stats-proto:
	cd $(STATS_DIR) && .venv/bin/python -m grpc_tools.protoc \
		-I. --python_out=. --grpc_python_out=. stats.proto
	PATH="$$PATH:$$(go env GOPATH)/bin" protoc \
		--proto_path=$(STATS_DIR) \
		--go_out=pkg/statsgrpc --go_opt=paths=source_relative \
		--go-grpc_out=pkg/statsgrpc --go-grpc_opt=paths=source_relative \
		$(STATS_DIR)/stats.proto

# Run Python stats engine tests
stats-test:
	cd $(STATS_DIR) && .venv/bin/python -m pytest test_engine.py -v

# Start the stats gRPC server (requires DB_* env vars)
stats-run:
	cd $(STATS_DIR) && .venv/bin/python server.py