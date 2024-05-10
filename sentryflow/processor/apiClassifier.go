// // SPDX-License-Identifier: Apache-2.0

package processor

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/5gsec/SentryFlow/config"
	"github.com/5gsec/SentryFlow/exporter"
	"github.com/5gsec/SentryFlow/protobuf"
	"google.golang.org/grpc"
)

// AH Local reference for AI Handler
var AH *AIHandler

// AIHandler Structure
type AIHandler struct {
	stopChan chan struct{}

	aggregatedLogs chan []*protobuf.APILog
	APIs           chan []string

	AIStream *streamInform
}

// streamInform Structure
type streamInform struct {
	AIStream protobuf.APIClassification_ClassifyAPIsClient
}

// init Function
func init() {
	AH = NewAIHandler()
}

// NewAIHandler Function
func NewAIHandler() *AIHandler {
	ah := &AIHandler{
		stopChan: make(chan struct{}),

		aggregatedLogs: make(chan []*protobuf.APILog),
		APIs:           make(chan []string),
	}

	return ah
}

// StartAPIClassifier Function
func StartAPIClassifier(wg *sync.WaitGroup) bool {
	AIEngineService := fmt.Sprintf("%s:%s", config.GlobalConfig.AIEngineService, config.GlobalConfig.AIEngineServicePort)

	// Set up a connection to the server.
	conn, err := grpc.Dial(AIEngineService, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("[AI] Could not connect: %v", err)
		return false
	}

	// Start serving gRPC server
	log.Printf("[AI] Successfully connected to %s for APIMetrics", AIEngineService)

	client := protobuf.NewAPIClassificationClient(conn)

	aiStream, err := client.ClassifyAPIs(context.Background())
	if err != nil {
		log.Fatalf("[AI] Could not make stream: %v", err)
		return false
	}

	AH.AIStream = &streamInform{
		AIStream: aiStream,
	}

	go sendAPIRoutine()
	go recvAPIRoutine()

	return true
}

// InsertAPILogsAI function
func InsertAPILogsAI(APIs []string) {
	AH.APIs <- APIs
}

// sendAPIRoutine Function
func sendAPIRoutine() {
	for {
		select {
		case aal, ok := <-AH.APIs:
			if !ok {
				log.Printf("[Exporter] EnvoyMetric exporter channel closed")
				return
			}

			curAPIRequest := &protobuf.APIClassificationRequest{
				API: aal,
			}

			err := AH.AIStream.AIStream.Send(curAPIRequest)
			if err != nil {
				log.Printf("[Exporter] AI Engine APIs exporting failed %v:", err)
			}
		case <-AH.stopChan:
			return
		}
	}
}

// recvAPIRoutine Function
func recvAPIRoutine() error {
	for {
		select {
		default:
			event, err := AH.AIStream.AIStream.Recv()
			APIMetrics := make(map[string]uint64)
			if err == io.EOF {
				return nil
			}

			if err != nil {
				log.Printf("[Envoy] Something went on wrong when receiving event: %v", err)
				return err
			}

			for api, count := range event.APIs {
				APIMetrics[api] = count
			}

			err = exporter.ExpH.SendAPIMetrics(&protobuf.APIMetrics{PerAPICounts: APIMetrics})
			if err != nil {
				log.Printf("[Envoy] Something went on wrong when Send API Metrics: %v", err)
				return err
			}
		case <-AH.stopChan:
			return nil
		}
	}
}
