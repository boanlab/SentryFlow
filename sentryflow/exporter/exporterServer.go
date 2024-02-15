// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"context"
	"log"
	metricAPI "sentryflow/metrics/api"
	"sentryflow/protobuf"
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
func (exs *Server) GetLog(param *protobuf.GetLogParam, stream protobuf.SentryFlow_GetLogServer) error {
	log.Printf("[Exporter] Client %s(%s) connected", param.Info.HostName, param.Info.IPAddress)

	curExporter := &Inform{
		stream:    stream,
		Hostname:  param.Info.HostName,
		IPAddress: param.Info.IPAddress,
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
func (exs *Server) GetAPIMetrics(_ context.Context, param *protobuf.GetAPIMetricsParam) (*protobuf.APIMetric, error) {
	log.Printf("[Exporter] Client %s(%s) connected", param.Info.HostName, param.Info.IPAddress)

	// Construct protobuf return value
	ret := protobuf.APIMetric{
		PerAPICounts: metricAPI.GetPerAPICount(),
	}

	return &ret, nil
}
