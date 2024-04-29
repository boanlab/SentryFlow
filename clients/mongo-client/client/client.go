// SPDX-License-Identifier: Apache-2.0

package client

import (
	pb "SentryFlow/protobuf"
	"context"
	"log"
	"mongo-client/mongodb"
)

// Feeder Structure
type Feeder struct {
	Running bool

	client            pb.SentryFlowClient
	logStream         pb.SentryFlow_GetAPILogClient
	envoyMetricStream pb.SentryFlow_GetEnvoyMetricsClient
	apiMetricStream   pb.SentryFlow_GetAPIMetricsClient

	dbHandler mongodb.MongoDBHandler

	Done chan struct{}
}

// NewClient Function
func NewClient(client pb.SentryFlowClient, clientInfo *pb.ClientInfo, logCfg string, metricCfg string, metricFilter string, mongoDBAddr string) *Feeder {
	fd := &Feeder{}

	fd.Running = true

	fd.client = client

	fd.Done = make(chan struct{})

	if logCfg != "none" {
		// Contact the server and print out its response
		logStream, err := client.GetAPILog(context.Background(), clientInfo)
		if err != nil {
			log.Fatalf("[Client] Could not get log: %v", err)
		}

		fd.logStream = logStream
	}

	if metricCfg != "none" && (metricFilter == "all" || metricFilter == "api") {
		amStream, err := client.GetAPIMetrics(context.Background(), clientInfo)
		if err != nil {
			log.Fatalf("[Client] Could not get log: %v", err)
		}

		fd.apiMetricStream = amStream
	}

	if metricCfg != "none" && (metricFilter == "all" || metricFilter == "envoy") {
		emStream, err := client.GetEnvoyMetrics(context.Background(), clientInfo)
		if err != nil {
			log.Fatalf("[Client] Could not get log: %v", err)
		}

		fd.envoyMetricStream = emStream
	}

	// Initialize DB
	dbHandler, err := mongodb.NewMongoDBHandler(mongoDBAddr)
	if err != nil {
		log.Fatalf("[MongoDB] Unable to intialize DB: %v", err)
	}
	fd.dbHandler = *dbHandler

	return fd
}

// LogRoutine Function
func (fd *Feeder) LogRoutine(logCfg string) {
	for fd.Running {
		select {
		default:
			data, err := fd.logStream.Recv()
			if err != nil {
				log.Fatalf("[Client] Failed to receive a log: %v", err)
				break
			}
			err = fd.dbHandler.InsertAPILog(data)
			if err != nil {
				log.Fatalf("[MongoDB] Failed to insert API Log: %v", err)
			}
		case <-fd.Done:
			return
		}
	}
}

// APIMetricRoutine Function
func (fd *Feeder) APIMetricRoutine(metricCfg string) {
	for fd.Running {
		select {
		default:
			data, err := fd.apiMetricStream.Recv()
			if err != nil {
				log.Fatalf("[Client] Failed to receive metrics: %v", err)
				break
			}
			err = fd.dbHandler.InsertAPIMetrics(data)
			if err != nil {
				log.Fatalf("[MongoDB] Failed to insert API Metrics: %v", err)
			}
		case <-fd.Done:
			return
		}
	}
}

// EnvoyMetricRoutine Function
func (fd *Feeder) EnvoyMetricRoutine(metricCfg string) {
	for fd.Running {
		select {
		default:
			data, err := fd.envoyMetricStream.Recv()
			if err != nil {
				log.Fatalf("[Client] Failed to receive metrics: %v", err)
				break
			}
			err = fd.dbHandler.InsertEnvoyMetrics(data)
			if err != nil {
				log.Fatalf("[MongoDB] Failed to insert Envoy Metrics: %v", err)
			}
		case <-fd.Done:
			return
		}
	}
}
