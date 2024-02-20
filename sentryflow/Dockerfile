# SPDX-License-Identifier: Apache-2.0

### Builder

FROM golang:1.19-alpine3.17 as builder

RUN apk --no-cache update
RUN apk add --no-cache git clang llvm make gcc protobuf make
RUN apk add --update alpine-sdk
RUN go install github.com/golang/protobuf/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

RUN mkdir /app
RUN mkdir /protobuf

WORKDIR /protobuf
COPY /protobuf .
RUN go mod tidy
RUN make build

WORKDIR /app
COPY /sentryflow .

RUN go mod tidy
RUN export CGO_ENABLED=1; export CC=gcc;
RUN go build -o sentryflow

### Make executable image

FROM alpine:3.18 as sentryflow

RUN echo "@community http://dl-cdn.alpinelinux.org/alpine/edge/community" | tee -a /etc/apk/repositories

RUN apk --no-cache update
RUN apk add bash

COPY --from=builder /app/sentryflow .

CMD ["./sentryflow"]
