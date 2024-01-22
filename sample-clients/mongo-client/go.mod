module mongo-client

go 1.19

replace otel-custom-collector/protobuf => ../../protobuf

require (
	github.com/google/uuid v1.3.1
	go.mongodb.org/mongo-driver v1.13.1
	google.golang.org/grpc v1.60.1
	otel-custom-collector/protobuf v0.0.0-00010101000000-000000000000
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/net v0.16.0 // indirect
	golang.org/x/sync v0.4.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
)
