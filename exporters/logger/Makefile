IMAGE_NAME = boanlab.com/client-stdout
TAG = v1

.PHONY: build

build:
	docker build -t $(IMAGE_NAME):$(TAG) -f ./Dockerfile ../../

.PHONY: clean

clean:
	docker rmi $(IMAGE_NAME):$(TAG)

.PHONY: run

run:
	docker run -it --rm $(IMAGE_NAME):$(TAG)
