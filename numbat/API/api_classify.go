package API

import protobuf "otel-custom-collector/protobuf"

type APINode struct {
	path  string
	count int
	child []*APINode
}

type classifiedAPI struct {
	destination string
	method      string
	URIRoot     *APINode
}

func classifyAPI(accessLogs []protobuf.AccessLog) { // return classifiedAPI
}
