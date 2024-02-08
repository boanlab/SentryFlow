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
	WgDaemon sync.WaitGroup
}

// NewNumbatDaemon Function
func NewNumbatDaemon() *NumbatDaemon {
	dm := new(NumbatDaemon)

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
		defer dm.WgDaemon.Done()
		K8s.RunInformers(StopChan)
	}()
}

// logProcessor Function
func (dm *NumbatDaemon) logProcessor() {
	dm.WgDaemon.Add(1)
	defer dm.WgDaemon.Done()

	// Start log processor
	go func() {
		defer dm.WgDaemon.Done()
		StartLogProcessor()
	}()

	log.Printf("[Numbat] Initialized log processor")
}

// metricAnalyzer Function
func (dm *NumbatDaemon) metricAnalyzer() {
	dm.WgDaemon.Add(1)

	// Initialize metrics analyzer
	go func() {
		defer dm.WgDaemon.Done()
		metrics.StartMetricsAnalyzer()
	}()

	log.Printf("[Numbat] Initialized metric analyzer")
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

	go func() {
		defer dm.WgDaemon.Done()
		err = Oh.StartOtelServer()
		if err != nil {
			log.Fatalf("[Numbat] Unable to start OpenTelemetry Server: %v", err)
			return
		}
		log.Printf("[Numbat] Initialized OpenTelemetry collector")
	}()
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

	go func() {
		defer dm.WgDaemon.Done()
		err = exporter.Exp.StartExporterServer()
		if err != nil {
			log.Fatalf("[Numbat] Unable to start Exporter Server: %v", err)
		}
		log.Printf("[Numbat] Initialized exporter")
	}()
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

	log.Printf("[Numbat] Successfully started Numbat")
	dm.WgDaemon.Wait()
}
