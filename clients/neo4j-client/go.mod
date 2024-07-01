module neo4j-client

go 1.21

replace SentryFlow/protobuf => ../../protobuf

require (
	SentryFlow/protobuf v0.0.0-00010101000000-000000000000
	github.com/neo4j/neo4j-go-driver/v4 v4.4.7
	google.golang.org/grpc v1.64.0
)

require (
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)
