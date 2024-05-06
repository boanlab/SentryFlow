// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// SentryFlowConfig structure
type SentryFlowConfig struct {
	CollectorAddr string // Address for Collector gRPC
	CollectorPort string // Port for Collector gRPC

	ExporterAddr string // IP address to use for exporter gRPC
	ExporterPort string // Port to use for exporter gRPC

	PatchingNamespaces           bool // Enable/Disable patching namespaces with 'istio-injection'
	RestartingPatchedDeployments bool // Enable/Disable restarting deployments after patching

	AggregationPeriod int // Period for aggregating metrics
	CleanUpPeriod     int // Period for cleaning up outdated metrics

	Debug bool // Enable/Disable SentryFlow debug mode
}

// GlobalConfig Global configuration for SentryFlow
var GlobalConfig SentryFlowConfig

// init Function
func init() {
	_ = LoadConfig()
}

// Config const
const (
	CollectorAddr string = "collectorAddr"
	CollectorPort string = "collectorPort"

	ExporterAddr string = "exporterAddr"
	ExporterPort string = "exporterPort"

	PatchingNamespaces           string = "patchingNamespaces"
	RestartingPatchedDeployments string = "restartingPatchedDeployments"

	AggregationPeriod string = "aggregationPeriod"
	CleanUpPeriod     string = "cleanUpPeriod"

	Debug string = "debug"
)

func readCmdLineParams() {
	collectorAddrStr := flag.String(CollectorAddr, "0.0.0.0", "Address for Collector gRPC")
	collectorPortStr := flag.String(CollectorPort, "4317", "Port for Collector gRPC")

	exporterAddrStr := flag.String(ExporterAddr, "0.0.0.0", "Address for Exporter gRPC")
	exporterPortStr := flag.String(ExporterPort, "8080", "Port for Exporter gRPC")

	patchingNamespacesB := flag.Bool(PatchingNamespaces, false, "Enable patching 'istio-injection' to all namespaces")
	restartingPatchedDeploymentsB := flag.Bool(RestartingPatchedDeployments, false, "Enable restarting the deployments in all patched namespaces")

	aggregationPeriodInt := flag.Int(AggregationPeriod, 1, "Period for aggregating metrics")
	cleanUpPeriodInt := flag.Int(CleanUpPeriod, 5, "Period for cleanning up outdated metrics")

	configDebugB := flag.Bool(Debug, false, "Enable debugging mode")

	var flags []string
	flag.VisitAll(func(f *flag.Flag) {
		kv := fmt.Sprintf("%s:%v", f.Name, f.Value)
		flags = append(flags, kv)
	})
	log.Printf("Arguments [%s]", strings.Join(flags, " "))

	flag.Parse()

	viper.SetDefault(CollectorAddr, *collectorAddrStr)
	viper.SetDefault(CollectorPort, *collectorPortStr)

	viper.SetDefault(ExporterAddr, *exporterAddrStr)
	viper.SetDefault(ExporterPort, *exporterPortStr)

	viper.SetDefault(PatchingNamespaces, *patchingNamespacesB)
	viper.SetDefault(RestartingPatchedDeployments, *restartingPatchedDeploymentsB)

	viper.SetDefault(AggregationPeriod, *aggregationPeriodInt)
	viper.SetDefault(CleanUpPeriod, *cleanUpPeriodInt)

	viper.SetDefault(Debug, *configDebugB)
}

// LoadConfig Load configuration
func LoadConfig() error {
	// Read configuration from command line
	readCmdLineParams()

	// Read environment variable, those are upper-cased
	viper.AutomaticEnv()

	GlobalConfig.CollectorAddr = viper.GetString(CollectorAddr)
	GlobalConfig.CollectorPort = viper.GetString(CollectorPort)

	GlobalConfig.ExporterAddr = viper.GetString(ExporterAddr)
	GlobalConfig.ExporterPort = viper.GetString(ExporterPort)

	GlobalConfig.PatchingNamespaces = viper.GetBool(PatchingNamespaces)
	GlobalConfig.RestartingPatchedDeployments = viper.GetBool(RestartingPatchedDeployments)

	GlobalConfig.AggregationPeriod = viper.GetInt(AggregationPeriod)
	GlobalConfig.CleanUpPeriod = viper.GetInt(CleanUpPeriod)

	GlobalConfig.Debug = viper.GetBool(Debug)

	log.Printf("Configuration [%+v]", GlobalConfig)

	return nil
}
