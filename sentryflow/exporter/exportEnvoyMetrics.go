// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"errors"
	"fmt"
	"log"

	"github.com/5gsec/SentryFlow/protobuf"
)

// == //

// envoyMetricsStreamInform structure
type envoyMetricsStreamInform struct {
	Hostname  string
	IPAddress string

	metricsStream protobuf.SentryFlow_GetEnvoyMetricsServer

	error chan error
}

// GetEnvoyMetrics Function (for gRPC)
func (exs *ExpService) GetEnvoyMetrics(info *protobuf.ClientInfo, stream protobuf.SentryFlow_GetEnvoyMetricsServer) error {
	log.Printf("[Exporter] Client %s (%s) connected (GetEnvoyMetrics)", info.HostName, info.IPAddress)

	currExporter := &envoyMetricsStreamInform{
		Hostname:      info.HostName,
		IPAddress:     info.IPAddress,
		metricsStream: stream,
	}

	ExpH.exporterLock.Lock()
	ExpH.envoyMetricsExporters = append(ExpH.envoyMetricsExporters, currExporter)
	ExpH.exporterLock.Unlock()

	return <-currExporter.error
}

// SendEnvoyMetrics Function
func (exp *ExpHandler) SendEnvoyMetrics(evyMetrics *protobuf.EnvoyMetrics) error {
	failed := 0
	total := len(exp.envoyMetricsExporters)

	for _, exporter := range exp.envoyMetricsExporters {
		if err := exporter.metricsStream.Send(evyMetrics); err != nil {
			log.Printf("[Exporter] Failed to export Envoy metrics to %s(%s): %v", exporter.Hostname, exporter.IPAddress, err)
			failed++
		}
	}

	if failed != 0 {
		msg := fmt.Sprintf("[Exporter] Failed to export Envoy metrics properly (%d/%d failed)", failed, total)
		return errors.New(msg)
	}

	return nil
}

// == //

// InsertEnvoyMetrics Function
func InsertEnvoyMetrics(evyMetrics *protobuf.EnvoyMetrics) {
	ExpH.exporterMetrics <- evyMetrics
}

// == //
