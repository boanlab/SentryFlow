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
	Count int
}

// StatsPerLabel structure
type StatsPerLabel struct {
	APIs        map[string]Stats
	LastUpdated uint64
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
	log.Printf("[Exporter] Client %s (%s) connected (GetAPIMetrics)", info.HostName, info.IPAddress)

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
	failed := 0
	total := len(exp.apiMetricsExporters)

	for _, exporter := range exp.apiMetricsExporters {
		if err := exporter.apiMetricsStream.Send(apiMetrics); err != nil {
			log.Printf("[Exporter] Failed to export API metrics to %s (%s): %v", exporter.Hostname, exporter.IPAddress, err)
			failed++
		}
	}

	if failed != 0 {
		msg := fmt.Sprintf("[Exporter] Failed to export API metrics properly (%d/%d failed)", failed, total)
		return errors.New(msg)
	}

	return nil
}

// == //

// UpdateStats Function
func UpdateStats(namespace string, label string, api string) {
	ExpH.statsPerLabelLock.RLock()
	defer ExpH.statsPerLabelLock.RUnlock()

	// Check if namespace+label exists
	if _, ok := ExpH.statsPerLabel[namespace+label]; !ok {
		ExpH.statsPerLabel[namespace+label] = StatsPerLabel{
			APIs:        make(map[string]Stats),
			LastUpdated: uint64(time.Now().Unix()),
		}
	}

	statsPerLabel := ExpH.statsPerLabel[namespace+label]
	statsPerLabel.LastUpdated = uint64(time.Now().Unix())

	// Check if API exists
	if _, ok := statsPerLabel.APIs[api]; !ok {
		init := Stats{
			Count: 1,
		}
		statsPerLabel.APIs[api] = init
	} else {
		stats := statsPerLabel.APIs[api]
		stats.Count++
		statsPerLabel.APIs[api] = stats
	}

	ExpH.statsPerLabel[namespace+label] = statsPerLabel
}

// AggregateAPIMetrics Function
func AggregateAPIMetrics() {
	ticker := time.NewTicker(time.Duration(config.GlobalConfig.AggregationPeriod) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ExpH.statsPerLabelLock.RLock()

			APIMetrics := make(map[string]uint64)

			for _, statsPerLabel := range ExpH.statsPerLabel {
				for api, stats := range statsPerLabel.APIs {
					APIMetrics[api] = uint64(stats.Count)
				}
			}

			if len(APIMetrics) > 0 {
				err := ExpH.SendAPIMetrics(&protobuf.APIMetrics{PerAPICounts: APIMetrics})
				if err != nil {
					log.Printf("[Envoy] Failed to export API metrics: %v", err)
					return
				}
			}

			ExpH.statsPerLabelLock.RUnlock()
		case <-ExpH.stopChan:
			return
		}
	}
}

// CleanUpOutdatedStats Function
func CleanUpOutdatedStats() {
	ticker := time.NewTicker(time.Duration(config.GlobalConfig.CleanUpPeriod) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ExpH.statsPerLabelLock.Lock()

			cleanUpTime := uint64((time.Now().Add(-time.Duration(config.GlobalConfig.CleanUpPeriod) * time.Second)).Unix())
			labelToDelete := []string{}

			for label, statsPerLabel := range ExpH.statsPerLabel {
				if statsPerLabel.LastUpdated < cleanUpTime {
					labelToDelete = append(labelToDelete, label)
				}
			}

			for _, label := range labelToDelete {
				delete(ExpH.statsPerLabel, label)
			}

			ExpH.statsPerLabelLock.Unlock()
		case <-ExpH.stopChan:
			return
		}
	}
}

// == //

// Exporting API metrics is handled by API Classifier

// == //
