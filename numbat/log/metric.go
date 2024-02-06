package log

import protobuf "numbat/protobuf"

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

func generateMetric(cal classifiedAPI) {

}

// Functions for generating metrics.
func statisticOfAPIsPerDestination(cal classifiedAPI) {

}

func statisticOfAPIsPerMin(cal classifiedAPI) {

}

func statisticOfErrorAPI(cal classifiedAPI) {

}

func statisticOfAPILatency(cal classifiedAPI) {

}

// plan to add features
