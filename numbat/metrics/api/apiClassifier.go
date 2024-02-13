// SPDX-License-Identifier: Apache-2.0

package api

type node struct {
	path  string
	count int
	child []*node
}

type classifiedAPI struct {
	destination string
	method      string
	URIRoot     *node
}

// classifyAPI Function
func classifyAPI(api string) {
}

// generateMetric Function
func generateMetric(cal classifiedAPI) {

}

// statisticOfAPIsPerDestination Function
func statisticOfAPIsPerDestination(cal classifiedAPI) {

}

// statisticOfAPIsPerMin Function
func statisticOfAPIsPerMin(cal classifiedAPI) {

}

// statisticOfErrorAPI Function
func statisticOfErrorAPI(cal classifiedAPI) {

}

// statisticOfAPILatency Function
func statisticOfAPILatency(cal classifiedAPI) {

}
