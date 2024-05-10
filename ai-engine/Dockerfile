# SPDX-License-Identifier: Apache-2.0

# Dockerfile
FROM ubuntu:22.04

RUN apt-get update
RUN apt-get -y install git wget python3 python3-pip
RUN pip3 install grpcio grpcio-tools

RUN mkdir -p /ai-engine/protobuf
WORKDIR /ai-engine

COPY /ai-engine .
COPY /protobuf ./protobuf

WORKDIR /ai-engine/stringlifier
RUN pip3 install .

WORKDIR /ai-engine
RUN pip install -r requirements.txt

CMD ["python3", "ai-engine.py"]
