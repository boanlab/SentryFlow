// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"log"

	"github.com/5GSEC/SentryFlow/protobuf"
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
	log.Printf("[Exporter] Client %s(%s) connected (GetLog)", info.HostName, info.IPAddress)

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

// GetEnvoyMetrics Function
func (exs *Server) GetEnvoyMetrics(info *protobuf.ClientInfo, stream protobuf.SentryFlow_GetEnvoyMetricsServer) error {
	log.Printf("[Exporter] Client %s(%s) connected (GetEnvoyMetrics)", info.HostName, info.IPAddress)

	curExporter := &metricStreamInform{
		metricStream: stream,
		Hostname:     info.HostName,
		IPAddress:    info.IPAddress,
	}

	// Append new exporter client for future use
	Exp.exporterLock.Lock()
	Exp.metricExporters = append(Exp.metricExporters, curExporter)
	Exp.exporterLock.Unlock()

	// Keeping gRPC stream alive
	// refer https://stackoverflow.com/questions/36921131/
	return <-curExporter.error
}

// GetAPIMetrics Function
func (exs *Server) GetAPIMetrics(info *protobuf.ClientInfo, stream protobuf.SentryFlow_GetAPIMetricsServer) error {
	log.Printf("[Exporter] Client %s(%s) connected (GetAPIMetrics)", info.HostName, info.IPAddress)

	curExporter := &apiMetricStreamInform{
		apiMetricStream: stream,
		Hostname:        info.HostName,
		IPAddress:       info.IPAddress,
	}

	Exp.exporterLock.Lock()
	Exp.apiMetricExporters = append(Exp.apiMetricExporters, curExporter)
	Exp.exporterLock.Unlock()

	return <-curExporter.error
}
