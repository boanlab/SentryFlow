// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"context"
	metricAPI "github.com/5GSEC/sentryflow/metrics/api"
	"github.com/5GSEC/sentryflow/protobuf"
	"log"
)

var exs *Server

// init Function
func init() {
	exs = NewExporterServer()
}

// Server Structure
type Server struct {
	protobuf.UnimplementedSentryFlowServer // @todo: make this fixed.
}

// NewExporterServer Function
func NewExporterServer() *Server {
	return new(Server)
}

// GetLog Function
func (exs *Server) GetLog(info *protobuf.ClientInfo, stream protobuf.SentryFlow_GetLogServer) error {
	log.Printf("[Exporter] Client %s(%s) connected", info.HostName, info.IPAddress)

	curExporter := &Inform{
		stream:    stream,
		Hostname:  info.HostName,
		IPAddress: info.IPAddress,
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
func (exs *Server) GetAPIMetrics(_ context.Context, info *protobuf.ClientInfo) (*protobuf.APIMetric, error) {
	log.Printf("[Exporter] Client %s(%s) connected", info.HostName, info.IPAddress)

	// Construct protobuf return value
	ret := protobuf.APIMetric{
		PerAPICounts: metricAPI.GetPerAPICount(),
	}

	return &ret, nil
}
