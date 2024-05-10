// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"errors"
	"log"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/json"
)

// meshConfig structure
type meshConfig struct {
	DefaultConfig struct {
		DiscoveryAddress      string `yaml:"discoveryAddress"`
		EnvoyAccessLogService struct {
			Address string `yaml:"address"`
		} `yaml:"envoyAccessLogService"`
		EnvoyMetricsService struct {
			Address string `yaml:"address"`
		} `yaml:"envoyMetricsService"`
	} `yaml:"defaultConfig"`

	DefaultProviders struct {
		AccessLogs []string `yaml:"accessLogs"`
		Metrics    []string `yaml:"metrics"`
	} `yaml:"defaultProviders"`

	EnableEnvoyAccessLogService bool `yaml:"enableEnvoyAccessLogService"`

	ExtensionProviders []struct {
		EnvoyOtelAls struct {
			Port    string `yaml:"port"`
			Service string `yaml:"service"`
		} `yaml:"envoyOtelAls"`
		Name string `yaml:"name"`
	} `yaml:"extensionProviders"`

	ExtraFields map[string]interface{} `yaml:",inline"` // all extra fields that SentryFlow will not touch
}

// PatchIstioConfigMap Function
func PatchIstioConfigMap() bool {
	log.Print("[PatchIstioConfigMap] Patching Istio ConfigMap")

	meshCfg, err := parseIstioConfigMap()
	if err != nil {
		log.Printf("[PatchIstioConfigMap] Unable to parse Istio ConfigMap: %v", err)
		return false
	}

	if isIstioAlreadyPatched(meshCfg) {
		log.Print("[PatchIstioConfigMap] Istio ConfigMap was already patched before, skipping...")
		return true
	}

	// set metrics and envoy access logging to Sentryflow
	meshCfg.DefaultConfig.EnvoyAccessLogService.Address = "sentryflow.sentryflow.svc.cluster.local:4317"
	meshCfg.DefaultConfig.EnvoyMetricsService.Address = "sentryflow.sentryflow.svc.cluster.local:4317"

	// add Sentryflow as Otel AL collector
	if patched, _ := isEnvoyOtelAlPatched(meshCfg); !patched {
		sfOtelAl := struct {
			EnvoyOtelAls struct {
				Port    string `yaml:"port"`
				Service string `yaml:"service"`
			} `yaml:"envoyOtelAls"`
			Name string `yaml:"name"`
		}{
			EnvoyOtelAls: struct {
				Port    string `yaml:"port"`
				Service string `yaml:"service"`
			}{
				Port:    "4317",
				Service: "sentryflow.sentryflow.svc.cluster.local",
			},
			Name: "sentryflow",
		}
		meshCfg.ExtensionProviders = append(meshCfg.ExtensionProviders, sfOtelAl)
	}

	// add default access log provider
	if patched, _ := isEnvoyALProviderPatched(meshCfg); !patched {
		meshCfg.DefaultProviders.AccessLogs = append(meshCfg.DefaultProviders.AccessLogs, "sentryflow")
	}

	meshCfg.EnableEnvoyAccessLogService = true

	yamlMeshCfg, err := yaml.Marshal(meshCfg)
	if err != nil {
		log.Printf("[PatchIstioConfigMap] Unable to unmarshall Istio ConfigMap: %v", err)
		return false
	}

	strMeshCfg := string(yamlMeshCfg[:])
	err = K8sH.updateConfigMap("istio-system", "istio", strMeshCfg)
	if err != nil {
		log.Printf("[PatchIstioConfigMap] Unable to update Istio ConfigMap: %v", err)
		return false
	}

	log.Print("[PatchIstioConfigMap] Successfully patched Istio ConfigMap")

	return true
}

// UnpatchIstioConfigMap Function
func UnpatchIstioConfigMap() bool {
	log.Print("[PatchIstioConfigMap] Unpatching Istio ConfigMap")

	meshCfg, err := parseIstioConfigMap()
	if err != nil {
		log.Printf("[PatchIstioConfigMap] Unable to parse Istio ConfigMap: %v", err)
		return false
	}

	// set metrics and envoy access logging back to empty value
	meshCfg.DefaultConfig.EnvoyAccessLogService.Address = ""
	meshCfg.DefaultConfig.EnvoyMetricsService.Address = ""

	// remove EnvoyOtelAl
	if patched, targetIdx := isEnvoyOtelAlPatched(meshCfg); patched {
		tmp := make([]struct {
			EnvoyOtelAls struct {
				Port    string `yaml:"port"`
				Service string `yaml:"service"`
			} `yaml:"envoyOtelAls"`
			Name string `yaml:"name"`
		}, 0)
		for idx, envoyOtelAl := range meshCfg.ExtensionProviders {
			if idx != targetIdx {
				tmp = append(tmp, envoyOtelAl)
			}
		}
		meshCfg.ExtensionProviders = tmp
	}

	// remove default access log provider
	if patched, targetIdx := isEnvoyALProviderPatched(meshCfg); patched {
		tmp := make([]string, 0)
		for idx, provider := range meshCfg.DefaultProviders.AccessLogs {
			if idx != targetIdx {
				tmp = append(tmp, provider)
			}
		}
		meshCfg.DefaultProviders.AccessLogs = tmp
	}

	// @todo this might be incorrect, the user might have just set up envoy access log service manually before.
	// @todo check if this shall actually be overwritten by SentryFlow
	// meshCfg.EnableEnvoyAccessLogService = false

	yamlMeshCfg, err := yaml.Marshal(meshCfg)
	if err != nil {
		log.Printf("[PatchIstioConfigMap] Unable to unmarshall Istio ConfigMap: %v", err)
		return false
	}

	strMeshCfg := string(yamlMeshCfg[:])
	err = K8sH.updateConfigMap("istio-system", "istio", strMeshCfg)
	if err != nil {
		log.Printf("[PatchIstioConfigMap] Unable to update Istio ConfigMap: %v", err)
		return false
	}

	log.Print("[PatchIstioConfigMap] Successfully unpatched Istio ConfigMap")

	return true
}

// parseIstioConfigMap Function
func parseIstioConfigMap() (meshConfig, error) {
	var meshCfg meshConfig

	configMapData, err := K8sH.getConfigMap("istio-system", "istio")
	if err != nil {
		return meshCfg, err
	}

	// unmarshall JSON format of Istio config
	var rawIstioCfg map[string]interface{}
	err = json.Unmarshal([]byte(configMapData), &rawIstioCfg)
	if err != nil {
		return meshCfg, err
	}

	// extract mesh field from configmap
	meshData, ok := rawIstioCfg["mesh"].(string)
	if !ok {
		return meshCfg, errors.New("[PatchIstioConfigMap] Unable to find field \"mesh\" from Istio config")
	}

	// unmarshall YAML format of Istio config
	err = yaml.Unmarshal([]byte(meshData), &meshCfg)
	if err != nil {
		return meshCfg, err
	}

	return meshCfg, nil
}

// isEnvoyOtelAlPatched Function
func isEnvoyOtelAlPatched(meshCfg meshConfig) (bool, int) {
	for idx, envoyOtelAl := range meshCfg.ExtensionProviders {
		if envoyOtelAl.Name == "sentryflow" &&
			envoyOtelAl.EnvoyOtelAls.Port == "4317" &&
			envoyOtelAl.EnvoyOtelAls.Service == "sentryflow.sentryflow.svc.cluster.local" {
			return true, idx
		}
	}

	return false, -1
}

// isEnvoyALProviderPatched Function
func isEnvoyALProviderPatched(meshCfg meshConfig) (bool, int) {
	for idx, accessLogProvider := range meshCfg.DefaultProviders.AccessLogs {
		if accessLogProvider == "sentryflow" {
			return true, idx
		}
	}
	return false, -1
}

// isIstioAlreadyPatched Function
func isIstioAlreadyPatched(meshCfg meshConfig) bool {
	if meshCfg.DefaultConfig.EnvoyAccessLogService.Address != "sentryflow.sentryflow.svc.cluster.local:4317" ||
		meshCfg.DefaultConfig.EnvoyMetricsService.Address != "sentryflow.sentryflow.svc.cluster.local:4317" {
		return false
	}

	if patched, _ := isEnvoyOtelAlPatched(meshCfg); !patched {
		return false
	}

	if patched, _ := isEnvoyALProviderPatched(meshCfg); !patched {
		return false
	}

	if !meshCfg.EnableEnvoyAccessLogService {
		return false
	}

	return true
}
