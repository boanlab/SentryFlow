# SPDX-License-Identifier: Apache-2.0

CLIENT_NAME = sentryflow-log-client
IMAGE_NAME = 5gsec/$(CLIENT_NAME)
TAG = v0.1

.PHONY: gofmt
gofmt:
	cd $(CURDIR); gofmt -w -s -d $(shell find . -type f -name '*.go' -print)

.PHONY: golint
golint:
ifeq (, $(shell which golint))
	@{ \
	set -e ;\
	GOLINT_TEMP_DIR=$$(mktemp -d) ;\
	cd $$GOLINT_TEMP_DIR ;\
	go mod init tmp ;\
	go get golang.org/x/lint/golint ;\
	go install golang.org/x/lint/golint ;\
	rm -rf $$GOLINT_TEMP_DIR ;\
	}
endif
	cd $(CURDIR); golint ./...

.PHONY: gosec
gosec:
ifeq (, $(shell which gosec))
	@{ \
	set -e ;\
	GOSEC_TEMP_DIR=$$(mktemp -d) ;\
	cd $$GOSEC_TEMP_DIR ;\
	go mod init tmp ;\
	go get github.com/securego/gosec/v2/cmd/gosec ;\
	go install github.com/securego/gosec/v2/cmd/gosec ;\
	rm -rf $$GOSEC_TEMP_DIR ;\
	}
endif
	cd $(CURDIR); gosec -exclude=G402 ./...

.PHONY: build gofmt golint gosec
build:
	go mod tidy
	go build -o $(CLIENT_NAME)

.PHONY: clean
clean:
	rm -f $(CLIENT_NAME)

.PHONY: build-image
build-image:
	docker build -t $(IMAGE_NAME):$(TAG) -f ./Dockerfile ../../

.PHONY: clean-image
clean-image:
	docker rmi $(IMAGE_NAME):$(TAG)

.PHONY: run-image
run-image:
	docker run -it --rm $(IMAGE_NAME):$(TAG)
