module relay-client

go 1.21.0

replace SentryFlow/protobuf => ../../protobuf

require (
	SentryFlow/protobuf v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.63.2
)

require (
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de // indirect
	google.golang.org/protobuf v1.34.0 // indirect
)