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

	dbHandler mongodb.DBHandler

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
			log.Fatalf("[Client] Could not get API log: %v", err)
		}

		fd.logStream = logStream
	}

	if metricCfg != "none" && (metricFilter == "all" || metricFilter == "api") {
		amStream, err := client.GetAPIMetrics(context.Background(), clientInfo)
		if err != nil {
			log.Fatalf("[Client] Could not get API metrics: %v", err)
		}

		fd.apiMetricStream = amStream
	}

	if metricCfg != "none" && (metricFilter == "all" || metricFilter == "envoy") {
		emStream, err := client.GetEnvoyMetrics(context.Background(), clientInfo)
		if err != nil {
			log.Fatalf("[Client] Could not get Envoy metrics: %v", err)
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

// APILogRoutine Function
func (fd *Feeder) APILogRoutine(logCfg string) {
	for fd.Running {
		select {
		default:
			data, err := fd.logStream.Recv()
			if err != nil {
				log.Fatalf("[Client] Failed to receive an API log: %v", err)
				break
			}
			err = fd.dbHandler.InsertAPILog(data)
			if err != nil {
				log.Fatalf("[MongoDB] Failed to insert an API log: %v", err)
			}
		case <-fd.Done:
			return
		}
	}
}

// APIMetricsRoutine Function
func (fd *Feeder) APIMetricsRoutine(metricCfg string) {
	for fd.Running {
		select {
		default:
			data, err := fd.apiMetricStream.Recv()
			if err != nil {
				log.Fatalf("[Client] Failed to receive API metrics: %v", err)
				break
			}
			err = fd.dbHandler.InsertAPIMetrics(data)
			if err != nil {
				log.Fatalf("[MongoDB] Failed to insert API metrics: %v", err)
			}
		case <-fd.Done:
			return
		}
	}
}

// EnvoyMetricsRoutine Function
func (fd *Feeder) EnvoyMetricsRoutine(metricCfg string) {
	for fd.Running {
		select {
		default:
			data, err := fd.envoyMetricStream.Recv()
			if err != nil {
				log.Fatalf("[Client] Failed to receive Envoy metrics: %v", err)
				break
			}
			err = fd.dbHandler.InsertEnvoyMetrics(data)
			if err != nil {
				log.Fatalf("[MongoDB] Failed to insert Envoy metrics: %v", err)
			}
		case <-fd.Done:
			return
		}
	}
}
