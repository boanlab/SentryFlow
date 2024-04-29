# SPDX-License-Identifier: Apache-2.0

### Builder

FROM golang:1.21-alpine3.17 as builder

RUN apk --no-cache update
RUN apk add --no-cache git clang llvm make gcc protobuf musl-dev
RUN apk add --update alpine-sdk

RUN go install github.com/golang/protobuf/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

RUN mkdir /sentryflow
RUN mkdir /protobuf

WORKDIR /protobuf
COPY /protobuf .

WORKDIR /sentryflow
COPY /sentryflow .

RUN export CGO_ENABLED=1; export CC=gcc;
RUN go build -o sentryflow

### Make executable image

FROM alpine:3.17 as sentryflow

# RUN echo "@community http://dl-cdn.alpinelinux.org/alpine/edge/community" | tee -a /etc/apk/repositories
# RUN apk --no-cache update
# RUN apk add bash

COPY --from=builder /sentryflow/sentryflow /

CMD ["/sentryflow"]
