// // SPDX-License-Identifier: Apache-2.0

package processor

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/5gsec/SentryFlow/config"
	"github.com/5gsec/SentryFlow/exporter"
	"github.com/5gsec/SentryFlow/protobuf"
	"google.golang.org/grpc"
)

// APIC Local reference for AI-driven API Classifier
var APIC *APIClassifier

// APIClassifier Structure
type APIClassifier struct {
	stopChan chan struct{}

	APIs chan []string

	connected   bool
	reConnTrial time.Duration

	AIStream *streamInform
}

// streamInform Structure
type streamInform struct {
	AIStream protobuf.APIClassifier_ClassifyAPIsClient
}

// init Function
func init() {
	APIC = NewAPIClassifier()
}

// NewAPIClassifier Function
func NewAPIClassifier() *APIClassifier {
	ah := &APIClassifier{
		stopChan: make(chan struct{}),

		APIs: make(chan []string),

		connected:   false,
		reConnTrial: (1 * time.Minute),
	}

	return ah
}

// initAPIClassifier Function
func initAPIClassifier() bool {
	AIEngineService := fmt.Sprintf("%s:%s", config.GlobalConfig.AIEngineService, config.GlobalConfig.AIEngineServicePort)

	// Set up a connection to the server
	conn, err := grpc.Dial(AIEngineService, grpc.WithInsecure())
	if err != nil {
		log.Printf("[APIClassifier] Failed to connect to %s: %v", AIEngineService, err)
		return false
	}

	log.Printf("[APIClassifier] Connecting to %s", AIEngineService)

	client := protobuf.NewAPIClassifierClient(conn)

	// Start serving gRPC server
	stream, err := client.ClassifyAPIs(context.Background())
	if err != nil {
		log.Printf("[APIClassifier] Failed to make a stream: %v", err)
		return false
	}

	log.Printf("[APIClassifier] Successfully connected to %s", AIEngineService)

	APIC.AIStream = &streamInform{
		AIStream: stream,
	}

	log.Print("[APIClassifier] Started API Classifier")

	return true
}

// StartAPIClassifier Function
func StartAPIClassifier(wg *sync.WaitGroup) bool {
	go connRoutine(wg)
	go sendAPIRoutine(wg)
	go recvAPIRoutine(wg)

	return true
}

// ClassifyAPIs function
func ClassifyAPIs(APIs []string) {
	if APIC.connected {
		APIC.APIs <- APIs
	}
}

// StopAPIClassifier Function
func StopAPIClassifier() bool {
	// one for connRoutine
	APIC.stopChan <- struct{}{}

	// one for sendAPIRoutine
	APIC.stopChan <- struct{}{}

	// one for recvAPIRoutine
	APIC.stopChan <- struct{}{}

	log.Print("[APIClassifier] Stopped API Classifier")

	return true
}

// connRoutine Function
func connRoutine(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		select {
		case <-APIC.stopChan:
			wg.Done()
			return
		default:
			if !APIC.connected {
				if initAPIClassifier() {
					APIC.connected = true
				} else {
					time.Sleep(APIC.reConnTrial)
				}
			}
		}
	}
}

// sendAPIRoutine Function
func sendAPIRoutine(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		if !APIC.connected {
			time.Sleep(APIC.reConnTrial)
			continue
		}

		select {
		case api, ok := <-APIC.APIs:
			if !ok {
				log.Print("[APIClassifier] Failed to fetch APIs from APIs channel")
				continue
			}

			curAPIRequest := &protobuf.APIClassifierRequest{
				API: api,
			}

			err := APIC.AIStream.AIStream.Send(curAPIRequest)
			if err != nil {
				log.Printf("[APIClassifier] Failed to send an API to AI Engine: %v", err)
				APIC.connected = false
				continue
			}
		case <-APIC.stopChan:
			wg.Done()
			return
		}
	}
}

// recvAPIRoutine Function
func recvAPIRoutine(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		if !APIC.connected {
			time.Sleep(APIC.reConnTrial)
			continue
		}

		select {
		default:
			APIMetrics := make(map[string]uint64)

			event, err := APIC.AIStream.AIStream.Recv()
			if err == io.EOF {
				continue
			} else if err != nil {
				log.Printf("[APIClassifier] Failed to receive an event from AI Engine: %v", err)
				APIC.connected = false
				continue
			}

			for api, count := range event.APIs {
				APIMetrics[api] = count
			}

			err = exporter.ExpH.SendAPIMetrics(&protobuf.APIMetrics{PerAPICounts: APIMetrics})
			if err != nil {
				log.Printf("[APIClassifier] Failed to export API metrics: %v", err)
				continue
			}
		case <-APIC.stopChan:
			wg.Done()
			return
		}
	}
}
