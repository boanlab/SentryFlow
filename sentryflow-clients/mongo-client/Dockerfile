# SPDX-License-Identifier: Apache-2.0

### Builder

FROM golang:1.19-alpine3.17 as builder

RUN apk --no-cache update
RUN apk add --no-cache git clang llvm make gcc protobuf

RUN mkdir /app
RUN mkdir /protobuf

WORKDIR /protobuf

COPY /protobuf .

WORKDIR /app

COPY /sentryflow-clients/mongo-client .

RUN go build -o mongo-client

### Make executable image

FROM alpine:3.18 as client

RUN echo "@community http://dl-cdn.alpinelinux.org/alpine/edge/community" | tee -a /etc/apk/repositories

RUN apk --no-cache update
RUN apk add bash

COPY --from=builder /app/mongo-client /

CMD ["/mongo-client"]
