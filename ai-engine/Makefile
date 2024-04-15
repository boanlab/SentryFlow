# SPDX-License-Identifier: Apache-2.0

ENGINE_NAME = sentryflow-ai-engine
IMAGE_NAME = 5gsec/$(ENGINE_NAME)
TAG = v0.1

.PHONY: build-image

build-image:
	docker build -t $(IMAGE_NAME):$(TAG) -f ./Dockerfile ../
