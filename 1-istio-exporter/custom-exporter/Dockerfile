FROM golang:1.20
RUN mkdir app
WORKDIR app
COPY . .
RUN go build
CMD ["./custom-exporter"]