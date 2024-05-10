# SPDX-License-Identifier: Apache-2.0

FROM python:3.10-bookworm

RUN mkdir -p /ai-engine/protobuf

RUN pip3 --no-cache-dir install grpcio grpcio-tools

WORKDIR /ai-engine

COPY /ai-engine .
COPY /protobuf ./protobuf

WORKDIR /ai-engine/stringlifier

RUN pip3 --no-cache-dir install .

WORKDIR /ai-engine

RUN pip3 --no-cache-dir install -r requirements.txt
RUN python3 -m grpc_tools.protoc --python_out=. --pyi_out=. --grpc_python_out=. -I=. protobuf/sentryflow_metrics.proto

CMD ["python3", "/ai-engine/classifier.py"]
