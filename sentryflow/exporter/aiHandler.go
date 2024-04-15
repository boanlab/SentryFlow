// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"context"
	"fmt"
	"io"
	"log"

	cfg "github.com/5GSEC/SentryFlow/config"
	"github.com/5GSEC/SentryFlow/protobuf"
	"github.com/5GSEC/SentryFlow/types"
	"google.golang.org/grpc"
)

// AH Local reference for AI handler server
var AH *aiHandler

// aiHandler Structure
type aiHandler struct {
	aiHost string
	aiPort string

	error          chan error
	stopChan       chan struct{}
	aggregatedLogs chan []*protobuf.APILog
	apis           chan []string

	aiStream *streamInform

	// @todo: add gRPC stream here for bidirectional connection
}

// streamInform Structure
type streamInform struct {
	aiStream protobuf.SentryFlowMetrics_GetAPIClassificationClient
}

// init Function
func init() {
	// Construct address and start listening
	AH = NewAIHandler(cfg.AIEngineService, cfg.AIEngineServicePort)
}

// NewAIHandler Function
func NewAIHandler(host string, port string) *aiHandler {
	ah := &aiHandler{
		aiHost: host,
		aiPort: port,

		stopChan:       make(chan struct{}),
		aggregatedLogs: make(chan []*protobuf.APILog),
		apis:           make(chan []string),
	}

	return ah
}

// initHandler Function
func (ah *aiHandler) InitAIHandler() bool {
	addr := fmt.Sprintf("%s:%s", "10.10.0.116", cfg.GlobalCfg.AIEngineServicePort)

	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
		return false
	}

	// Start serving gRPC server
	log.Printf("[gRPC] Successfully connected to %s for APIMetric", addr)

	client := protobuf.NewSentryFlowMetricsClient(conn)

	aiStream, err := client.GetAPIClassification(context.Background())

	AH.aiStream = &streamInform{
		aiStream: aiStream,
	}
	done := make(chan struct{})

	go sendAPIRoutine()
	go recvAPIRoutine(done)

	return true
}

// InsertAPILog function
func InsertAPILog(APIs []string) {
	AH.apis <- APIs
}

// callAI Function
func (ah *aiHandler) callAI(api string) error {
	// @todo: add gRPC send request
	return nil
}

// processBatch Function
func processBatch(batch []string, update bool) error {
	for range batch {

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

// sendAPIRoutine Function
func sendAPIRoutine() {
routineLoop:
	for {
		select {
		case aal, ok := <-AH.apis:
			if !ok {
				log.Printf("[Exporter] EnvoyMetric exporter channel closed")
				break routineLoop
			}

			curAPIRequest := &protobuf.APIClassificationRequest{
				Path: aal,
			}

			// err := AH.aiStream.Send(curAPIRequest)
			err := AH.aiStream.aiStream.Send(curAPIRequest)
			if err != nil {
				log.Printf("[Exporter] AI Engine APIs exporting failed %v:", err)
			}
		case <-AH.stopChan:
			break routineLoop
		}
	}

	return
}

// recvAPIRoutine Function
func recvAPIRoutine(done chan struct{}) error {
	for {
		select {
		default:
			event, err := AH.aiStream.aiStream.Recv()
			if err == io.EOF {
				return nil
			}

			if err != nil {
				log.Printf("[Envoy] Something went on wrong when receiving event: %v", err)
				return err
			}

			for key, value := range event.Fields {
				APICount := &types.PerAPICount{
					API:   key,
					Count: value,
				}
				err := MDB.PerAPICountInsert(APICount)
				if err != nil {
					log.Printf("unable to insert Classified API")
					return err
				}
			}
		case <-done:
			return nil
		}
	}
}
