// SPDX-License-Identifier: Apache-2.0

package processor

// import (
// 	"context"
// 	"fmt"
// 	"io"
// 	"log"

// 	"github.com/5gsec/SentryFlow/config"
// 	"github.com/5gsec/SentryFlow/protobuf"
// 	"github.com/5gsec/SentryFlow/types"
// 	"google.golang.org/grpc"
// )

// // AIH Local reference for AI handler server
// var AIH *AIHandler

// // AIHandler Structure
// type AIHandler struct {
// 	AIEngineAddr string
// 	AIEnginePort string

// 	error    chan error
// 	stopChan chan struct{}

// 	aggregatedLogs chan []*protobuf.APILog
// 	APIs           chan []string

// 	AIStream *streamInform
// }

// // streamInform Structure
// type streamInform struct {
// 	AIStream protobuf.SentryFlowMetrics_GetAPIClassificationClient
// }

// // init Function
// func init() {
// 	// Construct address and start listening
// 	ai = NewAIHandler(cfg.AIEngineAddr, cfg.AIEnginePort)
// }

// // NewAIHandler Function
// func NewAIHandler(addr string, port string) *AIHandler {
// 	ah := &AIHandler{
// 		AIEngineAddr: addr,
// 		AIEnginePort: port,

// 		stopChan: make(chan struct{}),

// 		aggregatedLogs: make(chan []*protobuf.APILog),
// 		APIs:           make(chan []string),
// 	}
// 	return ah
// }

// // initHandler Function
// func (ai *AIHandler) InitAIHandler() bool {
// 	AIEngineService := fmt.Sprintf("%s:%s", cfg.GlobalCfg.AIEngineAddr, cfg.GlobalCfg.AIEnginePort)

// 	// Set up a connection to the server.
// 	conn, err := grpc.Dial(AIEngineService, grpc.WithInsecure())
// 	if err != nil {
// 		log.Fatalf("[AI] Could not connect: %v", err)
// 		return false
// 	}

// 	// Start serving gRPC server
// 	log.Printf("[AI] Successfully connected to %s for APIMetrics", AIEngineService)

// 	client := protobuf.NewSentryFlowMetricsClient(conn)
// 	aiStream, err := client.GetAPIClassification(context.Background())

// 	ai.AIStream = &streamInform{
// 		AIStream: aiStream,
// 	}

// 	done := make(chan struct{})

// 	go sendAPIRoutine()
// 	go recvAPIRoutine(done)

// 	return true
// }

// // InsertAPILog function
// func InsertAPILog(APIs []string) {
// 	ai.APIs <- APIs
// }

// // callAI Function
// func (ah *aiHandler) callAI(api string) error {
// 	// @todo: add gRPC send request
// 	return nil
// }

// // processBatch Function
// func processBatch(batch []string, update bool) error {
// 	for range batch {

// 	}

// 	return nil
// }

// // performHealthCheck Function
// func (ah *aiHandler) performHealthCheck() error {
// 	return nil
// }

// // disconnect Function
// func (ah *aiHandler) disconnect() {
// 	return
// }

// // sendAPIRoutine Function
// func sendAPIRoutine() {
// routineLoop:
// 	for {
// 		select {
// 		case aal, ok := <-AH.apis:
// 			if !ok {
// 				log.Printf("[Exporter] EnvoyMetric exporter channel closed")
// 				break routineLoop
// 			}

// 			curAPIRequest := &protobuf.APIClassificationRequest{
// 				Path: aal,
// 			}

// 			// err := AH.aiStream.Send(curAPIRequest)
// 			err := AH.aiStream.aiStream.Send(curAPIRequest)
// 			if err != nil {
// 				log.Printf("[Exporter] AI Engine APIs exporting failed %v:", err)
// 			}
// 		case <-AH.stopChan:
// 			break routineLoop
// 		}
// 	}

// 	return
// }

// // recvAPIRoutine Function
// func recvAPIRoutine(done chan struct{}) error {
// 	for {
// 		select {
// 		default:
// 			event, err := AH.aiStream.aiStream.Recv()
// 			if err == io.EOF {
// 				return nil
// 			}

// 			if err != nil {
// 				log.Printf("[Envoy] Something went on wrong when receiving event: %v", err)
// 				return err
// 			}

// 			for key, value := range event.Fields {
// 				APICount := &types.PerAPICount{
// 					API:   key,
// 					Count: value,
// 				}
// 				err := MDB.PerAPICountInsert(APICount)
// 				if err != nil {
// 					log.Printf("unable to insert Classified API")
// 					return err
// 				}
// 			}
// 		case <-done:
// 			return nil
// 		}
// 	}
// }
