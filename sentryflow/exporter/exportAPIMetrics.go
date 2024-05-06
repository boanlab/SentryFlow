// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/5gsec/SentryFlow/config"
	"github.com/5gsec/SentryFlow/protobuf"
)

// == //

// Stats Structure
type Stats struct {
	Count      int
	LastUpdate uint64
}

// StatsPerNamespace structure
type StatsPerNamespace struct {
	APIs map[string]Stats
}

// StatsPerLabel structure
type StatsPerLabel struct {
	APIs map[string]Stats
}

// == //

// apiMetricStreamInform structure
type apiMetricStreamInform struct {
	Hostname  string
	IPAddress string

	apiMetricsStream protobuf.SentryFlow_GetAPIMetricsServer

	error chan error
}

// GetAPIMetrics Function (for gRPC)
func (exs *ExpService) GetAPIMetrics(info *protobuf.ClientInfo, stream protobuf.SentryFlow_GetAPIMetricsServer) error {
	log.Printf("[Exporter] Client %s(%s) connected (GetAPIMetrics)", info.HostName, info.IPAddress)

	currExporter := &apiMetricStreamInform{
		Hostname:         info.HostName,
		IPAddress:        info.IPAddress,
		apiMetricsStream: stream,
	}

	ExpH.exporterLock.Lock()
	ExpH.apiMetricsExporters = append(ExpH.apiMetricsExporters, currExporter)
	ExpH.exporterLock.Unlock()

	return <-currExporter.error
}

// SendAPIMetrics Function
func (exp *ExpHandler) SendAPIMetrics(apiMetrics *protobuf.APIMetrics) error {
	var err error

	failed := 0
	total := len(exp.apiMetricsExporters)

	for _, exporter := range exp.apiMetricsExporters {
		currRetry := 0
		maxRetry := 3

		for currRetry < maxRetry {
			if err = exporter.apiMetricsStream.Send(apiMetrics); err != nil {
				log.Printf("[Exporter] Unable to send API Metrics to %s(%s) retry=%d/%d: %v", exporter.Hostname, exporter.IPAddress, currRetry, maxRetry, err)
				currRetry++
			} else {
				break
			}
		}

		if err != nil {
			failed++
		}
	}

	if failed != 0 {
		msg := fmt.Sprintf("unable to send API Metrics properly %d/%d failed", failed, total)
		return errors.New(msg)
	}

	return nil
}

// == //

// UpdateStats Function
func UpdateStats(namespace string, label string, api string) {
	// == //

	ExpH.statsPerNamespaceLock.Lock()

	// Check if namespace exists
	if _, ok := ExpH.statsPerNamespace[namespace]; !ok {
		ExpH.statsPerNamespace[namespace] = StatsPerNamespace{
			APIs: make(map[string]Stats),
		}
	}

	statsPerNamespace := ExpH.statsPerNamespace[namespace]

	// Check if API exists
	if _, ok := statsPerNamespace.APIs[api]; !ok {
		init := Stats{
			Count:      1,
			LastUpdate: uint64(time.Now().Unix()),
		}
		statsPerNamespace.APIs[api] = init
	} else {
		stats := statsPerNamespace.APIs[api]

		stats.Count++
		stats.LastUpdate = uint64(time.Now().Unix())

		statsPerNamespace.APIs[api] = stats
	}

	ExpH.statsPerNamespace[namespace] = statsPerNamespace

	ExpH.statsPerNamespaceLock.Unlock()

	// == //

	ExpH.statsPerLabelLock.Lock()

	// Check if namespace+label exists
	if _, ok := ExpH.statsPerLabel[namespace+label]; !ok {
		ExpH.statsPerLabel[namespace+label] = StatsPerLabel{
			APIs: make(map[string]Stats),
		}
	}

	statsPerLabel := ExpH.statsPerLabel[namespace+label]

	// Check if API exists
	if _, ok := statsPerLabel.APIs[api]; !ok {
		init := Stats{
			Count:      1,
			LastUpdate: uint64(time.Now().Unix()),
		}
		statsPerLabel.APIs[api] = init
	} else {
		stats := statsPerLabel.APIs[api]

		stats.Count++
		stats.LastUpdate = uint64(time.Now().Unix())

		statsPerLabel.APIs[api] = stats
	}

	ExpH.statsPerLabel[namespace+label] = statsPerLabel

	ExpH.statsPerLabelLock.Unlock()

	// == //
}

// AggregateAPIMetrics Function
func AggregateAPIMetrics() {
	ticker := time.NewTicker(time.Duration(config.GlobalConfig.AggregationPeriod) * time.Second)
	defer ticker.Stop()

	for {
		select {
		//
		}
	}
}

// CleanUpOutdatedStats Function
func CleanUpOutdatedStats() {
	ticker := time.NewTicker(time.Duration(config.GlobalConfig.CleanUpPeriod) * time.Second)
	defer ticker.Stop()

	for {
		select {
		//
		}
	}
}

// == //
