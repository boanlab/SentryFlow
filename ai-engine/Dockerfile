# SPDX-License-Identifier: Apache-2.0

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
RUN python3 -m grpc_tools.protoc --python_out=. --pyi_out=. --grpc_python_out=. -I=. protobuf/sentryflow_metrics.proto

RUN pip install -r requirements.txt

CMD ["python3", "./ai_engine.py"]
