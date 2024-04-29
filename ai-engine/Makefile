# SPDX-License-Identifier: Apache-2.0

ENGINE_NAME = sentryflow-ai-engine
IMAGE_NAME = 5gsec/$(ENGINE_NAME)
TAG = v0.1

.PHONY: build-image
build-image:
	docker build -t $(IMAGE_NAME):$(TAG) -f ./Dockerfile ../

.PHONY: clean-image
clean-image:
	docker rmi $(IMAGE_NAME):$(TAG)

.PHONY: run-image
run-image:
	docker run -it --rm $(IMAGE_NAME):$(TAG)
