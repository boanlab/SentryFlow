// SPDX-License-Identifier: Apache-2.0

package api

// ah Local reference for AI handler server
var ah *aiHandler

// init Function
func init() {

}

// aiHandler Structure
type aiHandler struct {
	aiHost string
	aiPort string

	// @todo: add gRPC stream here for bidirectional connection
}

// newAIHandler Function
func newAIHandler(host string, port string) *aiHandler {
	ah := &aiHandler{
		aiHost: host,
		aiPort: port,
	}

	return ah
}

// initHandler Function
func (ah *aiHandler) initHandler() error {
	return nil
}

// callAI Function
func (ah *aiHandler) callAI(api string) error {
	// @todo: add gRPC send request
	return nil
}

// processBatch Function
func processBatch(batch []string, update bool) error {
	for _, _ = range batch {

	}

	return nil
}

// performHealthCheck Function
func (ah *aiHandler) performHealthCheck() error {
	return nil
}

// disconnect Function
func (ah *aiHandler) disconnect() {
	return
}
