package core

import (
	"log"
	cfg "numbat/config"
)

// StopChan Channel
var StopChan chan struct{}

// init Function
func init() {
	StopChan = make(chan struct{})
}

// NumbatDaemon Structure
type NumbatDaemon struct {
	K8sEnabled bool
}

// NewNumbatDaemon Function
func NewNumbatDaemon() *NumbatDaemon {
	dm := new(NumbatDaemon)

	dm.K8sEnabled = false

	return dm
}

// DestroyNumbatDaemon Function
func (dm *NumbatDaemon) DestroyNumbatDaemon() {

}

// WatchK8s Function
func (dm *NumbatDaemon) WatchK8s() {
	K8s.RunInformers(StopChan)
}

// PatchK8s Function
func (dm *NumbatDaemon) PatchK8s() error {
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

	go dm.WatchK8s()
	log.Printf("[Numbat] Started to monitor Kubernetes resources")

	if dm.PatchK8s() != nil {
		log.Printf("[Numbat] Failed to patch Kubernetes")
	}
	log.Printf("[Numbat] Patched Kubernetes and Istio configuration")
}
