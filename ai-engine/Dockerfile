# SPDX-License-Identifier: Apache-2.0

# Dockerfile
FROM ubuntu:latest

RUN apt-get update && apt-get -y install python3 python3-pip wget git

RUN git clone https://github.com/isu-kim/stringlifier.git
WORKDIR ./stringlifier
RUN pip install .

RUN mkdir /app
WORKDIR /app
COPY /ai-engine .

# Build protobuf for Python
RUN pip install grpcio grpcio-tools
RUN mkdir protobuf/
COPY /protobuf ./protobuf

# Due to python import bugs, we have to compile protoc using this command
# Refer to https://github.com/protocolbuffers/protobuf/issues/1491#issuecomment-261621112 for more information on this
RUN python3 -m grpc_tools.protoc --python_out=. --pyi_out=. --grpc_python_out=. -I=. protobuf/sentryflow_metrics.proto

WORKDIR /app
RUN pip install -r requirements.txt

CMD ["python3", "ai-engine.py"]
