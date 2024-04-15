PROTO:=sentryflow.proto sentryflow_metrics.proto
PB_GO:=$(PROTO:.proto=.pb.go)

.PHONY: build
build: $(PB_GO) go.sum

%.pb.go: %.proto
	go mod tidy
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative,require_unimplemented_servers=false $<

go.sum: go.mod
	go get .

.PHONY: clean
clean:
	rm -f go.sum *.pb.go

.PHONY: deps
deps:
	sudo cp bin/protoc /usr/local/bin/
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
