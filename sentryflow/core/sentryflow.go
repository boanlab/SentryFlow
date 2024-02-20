// SPDX-License-Identifier: Apache-2.0

package core

import (
	cfg "github.com/5GSEC/sentryflow/config"
	"github.com/5GSEC/sentryflow/exporter"
	"github.com/5GSEC/sentryflow/metrics"
	"log"
	"sync"
)

// StopChan Channel
var StopChan chan struct{}

// init Function
func init() {
	StopChan = make(chan struct{})
}

// SentryFlowDaemon Structure
type SentryFlowDaemon struct {
	WgDaemon *sync.WaitGroup
}

// NewSentryFlowDaemon Function
func NewSentryFlowDaemon() *SentryFlowDaemon {
	dm := new(SentryFlowDaemon)

	dm.WgDaemon = new(sync.WaitGroup)

	return dm
}

// DestroySentryFlowDaemon Function
func (dm *SentryFlowDaemon) DestroySentryFlowDaemon() {

}

// watchK8s Function
func (dm *SentryFlowDaemon) watchK8s() {
	K8s.RunInformers(StopChan, dm.WgDaemon)
}

// logProcessor Function
func (dm *SentryFlowDaemon) logProcessor() {
	StartLogProcessor(dm.WgDaemon)
	log.Printf("[SentryFlow] Started log processor")
}

// metricAnalyzer Function
func (dm *SentryFlowDaemon) metricAnalyzer() {
	metrics.StartMetricsAnalyzer(dm.WgDaemon)
	log.Printf("[SentryFlow] Started metric analyzer")
}

// exporterServer Function
func (dm *SentryFlowDaemon) exporterServer() {
	// Initialize and start exporter server
	err := exporter.Exp.InitExporterServer()
	if err != nil {
		log.Fatalf("[SentryFlow] Unable to initialize Exporter Server: %v", err)
		return
	}

	err = exporter.Exp.StartExporterServer(dm.WgDaemon)
	if err != nil {
		log.Fatalf("[SentryFlow] Unable to start Exporter Server: %v", err)
	}
	log.Printf("[SentryFlow] Initialized exporter")
}

// patchK8s Function
func (dm *SentryFlowDaemon) patchK8s() error {
	err := K8s.PatchIstioConfigMap()
	if err != nil {
		return err
	}

	if cfg.GlobalCfg.PatchNamespace {
		err = K8s.PatchNamespaces()
		if err != nil {
			return err
		}
	}

	if cfg.GlobalCfg.PatchRestartDeployments {
		err = K8s.PatchRestartDeployments()
		if err != nil {
			return err
		}
	}

	return nil
}

// SentryFlow Function
func SentryFlow() {
	// create a daemon
	dm := NewSentryFlowDaemon()

	// Initialize Kubernetes client
	if !K8s.InitK8sClient() {
		log.Printf("[Error] Failed to initialize Kubernetes client")
		dm.DestroySentryFlowDaemon()
		return
	}

	log.Printf("[SentryFlow] Initialized Kubernetes client")

	dm.watchK8s()
	log.Printf("[SentryFlow] Started to monitor Kubernetes resources")

	if dm.patchK8s() != nil {
		log.Printf("[SentryFlow] Failed to patch Kubernetes")
	}
	log.Printf("[SentryFlow] Patched Kubernetes and Istio configuration")

	if !MDB.InitMetricsDBHandler() {
		log.Printf("[Error] Failed to initialize Metrics DB")
		dm.DestroySentryFlowDaemon()
		return
	}
	log.Printf("[SentryFlow] Successfuly initialized metrics DB")

	// Start log processor
	dm.logProcessor()

	// Start metric analyzer
	dm.metricAnalyzer()

	// Start exporter server
	dm.exporterServer()

	log.Printf("[SentryFlow] Successfully started SentryFlow")
	dm.WgDaemon.Wait()
}
