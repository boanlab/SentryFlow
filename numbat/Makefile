IMAGE_NAME = boanlab/numbat
TAG = v0.0.1

.PHONY: build

build:
	docker build -t $(IMAGE_NAME):$(TAG) -f ./Dockerfile ../

.PHONY: clean

clean:
	docker rmi $(IMAGE_NAME):$(TAG)
