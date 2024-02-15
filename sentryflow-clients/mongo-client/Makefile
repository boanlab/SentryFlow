# SPDX-License-Identifier: Apache-2.0

IMAGE_NAME = 5gsec/sentryflow-mongo-client
TAG = v0.1

.PHONY: build

build:
	docker build -t $(IMAGE_NAME):$(TAG) -f ./Dockerfile ../../

.PHONY: clean

clean:
	docker rmi $(IMAGE_NAME):$(TAG)

.PHONY: run

run:
	docker run -it --rm $(IMAGE_NAME):$(TAG)
