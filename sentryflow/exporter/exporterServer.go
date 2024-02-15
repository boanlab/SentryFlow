// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"context"
	"log"
	metricAPI "numbat/metrics/api"
	"numbat/protobuf"
)

var exs *Server

// init Function
func init() {
	exs = NewExporterServer()
}

// Server Structure
type Server struct {
	protobuf.UnimplementedNumbatServer
}

// NewExporterServer Function
func NewExporterServer() *Server {
	return new(Server)
}

// GetLog Function
func (exs *Server) GetLog(client *protobuf.ClientInfo, stream protobuf.Numbat_GetLogServer) error {
	log.Printf("[Exporter] Client %s(%s) connected", client.Hostname, client.Hostname)

	curExporter := &Inform{
		stream:    stream,
		Hostname:  client.Hostname,
		IPAddress: client.IpAddress,
	}

	// Append new exporter client for future use
	Exp.exporterLock.Lock()
	Exp.exporters = append(Exp.exporters, curExporter)
	Exp.exporterLock.Unlock()

	// Keeping gRPC stream alive
	// refer https://stackoverflow.com/questions/36921131/
	return <-curExporter.error
}

// GetAPIMetrics Function
func (exs *Server) GetAPIMetrics(_ context.Context, client *protobuf.ClientInfo) (*protobuf.APIMetric, error) {
	log.Printf("[Exporter] Client %s(%s) connected", client.Hostname, client.Hostname)

	// Construct protobuf return value
	ret := protobuf.APIMetric{
		PerAPICounts: metricAPI.GetPerAPICount(),
	}

	return &ret, nil
}
