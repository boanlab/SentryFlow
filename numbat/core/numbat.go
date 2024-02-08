package core

import (
	"log"
	cfg "numbat/config"
	"numbat/exporter"
	"numbat/metrics"
	"sync"
)

// StopChan Channel
var StopChan chan struct{}

// init Function
func init() {
	StopChan = make(chan struct{})
}

// NumbatDaemon Structure
type NumbatDaemon struct {
	K8sEnabled            bool
	ExporterEnabled       bool
	OtelServerEnabled     bool
	LogProcessorEnabled   bool
	MetricAnalyzerEnabled bool

	WgDaemon sync.WaitGroup
}

// NewNumbatDaemon Function
func NewNumbatDaemon() *NumbatDaemon {
	dm := new(NumbatDaemon)

	dm.K8sEnabled = false
	dm.ExporterEnabled = false
	dm.OtelServerEnabled = false
	dm.LogProcessorEnabled = false
	dm.MetricAnalyzerEnabled = false

	dm.WgDaemon = sync.WaitGroup{}

	return dm
}

// DestroyNumbatDaemon Function
func (dm *NumbatDaemon) DestroyNumbatDaemon() {

}

// watchK8s Function
func (dm *NumbatDaemon) watchK8s() {
	dm.WgDaemon.Add(1)
	go func() {
		K8s.RunInformers(StopChan)
	}()
	dm.WgDaemon.Done()
}

// logProcessor Function
func (dm *NumbatDaemon) logProcessor() {
	dm.WgDaemon.Add(1)

	// Initialize log processor
	StartLogProcessor()
	dm.LogProcessorEnabled = true
	log.Printf("[Numbat] Initialized log processor")

	dm.WgDaemon.Done()
}

// metricAnalyzer Function
func (dm *NumbatDaemon) metricAnalyzer() {
	dm.WgDaemon.Add(1)

	// Initialize metrics analyzer
	metrics.StartMetricsAnalyzer()
	dm.MetricAnalyzerEnabled = true
	log.Printf("[Numbat] Initialized metric analyzer")

	dm.WgDaemon.Done()
}

// otelServer Function
func (dm *NumbatDaemon) otelServer() {
	dm.WgDaemon.Add(1)

	// Initialize and start OpenTelemetry Server
	err := Oh.InitOtelServer()
	if err != nil {
		log.Fatalf("[Numbat] Unable to intialize OpenTelemetry Server: %v", err)
		return
	}

	err = Oh.StartOtelServer()
	if err != nil {
		log.Fatalf("[Numbat] Unable to start OpenTelemetry Server: %v", err)
		return
	}
	dm.OtelServerEnabled = true
	log.Printf("[Numbat] Initialized OpenTelemetry collector")

	dm.WgDaemon.Done()
}

// exporterServer Function
func (dm *NumbatDaemon) exporterServer() {
	dm.WgDaemon.Add(1)

	// Initialize and start exporter server
	err := exporter.Exp.InitExporterServer()
	if err != nil {
		log.Fatalf("[Numbat] Unable to initialize Exporter Server: %v", err)
		return
	}

	err = exporter.Exp.StartExporterServer()
	if err != nil {
		log.Fatalf("[Numbat] Unable to start Exporter Server: %v", err)
	}
	dm.ExporterEnabled = true
	log.Printf("[Numbat] Initialized exporter")

	dm.WgDaemon.Done()
}

// patchK8s Function
func (dm *NumbatDaemon) patchK8s() error {
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

// Numbat Function
func Numbat() {
	// create a daemon
	dm := NewNumbatDaemon()

	// Initialize Kubernetes client
	if !K8s.InitK8sClient() {
		log.Printf("[Error] Failed to initialize Kubernetes client")
		dm.DestroyNumbatDaemon()
		return
	}

	log.Printf("[Numbat] Initialized Kubernetes client")
	dm.K8sEnabled = true

	dm.watchK8s()
	log.Printf("[Numbat] Started to monitor Kubernetes resources")

	if dm.patchK8s() != nil {
		log.Printf("[Numbat] Failed to patch Kubernetes")
	}
	log.Printf("[Numbat] Patched Kubernetes and Istio configuration")

	// Start log processor
	dm.logProcessor()

	// Start metric analyzer
	dm.metricAnalyzer()

	// Start OpenTelemetry server
	dm.otelServer()

	// Start exporter server
	dm.exporterServer()

	if dm.K8sEnabled && dm.OtelServerEnabled && dm.ExporterEnabled &&
		dm.LogProcessorEnabled && dm.MetricAnalyzerEnabled {
		log.Printf("[Numbat] Successfully started Numbat")
	} else {
		log.Fatalf("[Numbat] Unable to start Numbat successfully")
	}
}
