PROTO:=sentryflow.proto sentryflow_metrics.proto
PBGO:=$(PROTO:.proto=.pb.go)

.PHONY: build
build: $(PBGO) go.sum

go.sum: go.mod
	go get .

%.pb.go: %.proto
	go mod tidy
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative,require_unimplemented_servers=false $<

.PHONY: clean
clean:
	rm -f go.sum *.pb.go