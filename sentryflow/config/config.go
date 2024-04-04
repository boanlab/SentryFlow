// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// SentryFlowConfig structure
type SentryFlowConfig struct {
	OtelGRPCListenAddr string // IP address to use for OTEL gRPC
	OtelGRPCListenPort string // Port to use for OTEL gRPC

	CustomExportListenAddr string // IP address to use for custom exporter gRPC
	CustomExportListenPort string // Port to use for custom exporter gRPC

	PatchNamespace          bool // Enable/Disable patching namespace for Istio injection
	PatchRestartDeployments bool // Enable/Disable restarting deployments after patching

	AIEngineService     string
	AIEngineServicePort string
	AIEngineBatchSize   int

	MetricsDBFileName        string // String value of MetricsDB file (sqlite3 db file)
	MetricsDBAggregationTime int    // Value of APILog Aggregation Time
	MetricsDBClearTime       int    // Value of APIMetric DB Clear time
	APIMetricsSendTime       int    // Value of APIMetric send time

	CollectorEnableOpenTelemetry bool // Enable/Disable OpenTelemetry Collector
	Debug                        bool // Enable/Disable SentryFlow debug mode
}

// GlobalCfg Global configuration for SentryFlow
var GlobalCfg SentryFlowConfig

// init Function
func init() {
	_ = LoadConfig()
}

// Config const
const (
	OtelGRPCListenAddr           string = "otelGRPCListenAddr"
	OtelGRPCListenPort           string = "otelGRPCListenPort"
	CustomExportListenAddr       string = "customExportListenAddr"
	CustomExportListenPort       string = "customExportListenPort"
	PatchNamespace               string = "patchNamespace"
	PatchRestartDeployments      string = "patchRestartDeployments"
	AIEngineService              string = "aiEngineService"
	AIEngineServicePort          string = "aiEngineServicePort"
	AIEngineBatchSize            string = "aiEngineBatchSize"
	MetricsDBFileName            string = "metricsDBFileName"
	MetricsDBAggregationTime     string = "metricsDBAggregationTime"
	MetricsDBClearTime           string = "metricsDBClearTime"
	APIMetricsSendTime           string = "apiMetricsSendTime"
	CollectorEnableOpenTelemetry string = "collectorEnableOpenTelemetry"
	Debug                        string = "debug"
)

func readCmdLineParams() {
	otelGRPCListenAddrStr := flag.String(OtelGRPCListenAddr, "0.0.0.0", "OTEL gRPC server listen address")
	otelGRPCListenPortStr := flag.String(OtelGRPCListenPort, "4317", "OTEL gRPC server listen port")
	customExportListenAddrStr := flag.String(CustomExportListenAddr, "0.0.0.0", "Custom export gRPC server listen address")
	customExportListenPortStr := flag.String(CustomExportListenPort, "8080", "Custom export gRPC server listen port")
	patchNamespaceB := flag.Bool(PatchNamespace, false, "Enable/Disable patching Istio injection to all namespaces")
	patchRestartDeploymentsB := flag.Bool(PatchRestartDeployments, false, "Enable/Disable restarting deployments in all namespaces")
	aiEngineServiceStr := flag.String(AIEngineService, "ai-engine.sentryflow.svc.cluster.local", "Service address for SentryFlow AI Engine")
	aiEngineServicePortStr := flag.String(AIEngineServicePort, "5000", "Service Port for SentryFlow AI Engine")
	aiEngineBatchSizeInt := flag.Int(AIEngineBatchSize, 5, "Batch size for SentryFlow AI Engine")
	metricsDBFileNameStr := flag.String(MetricsDBFileName, "/etc/sentryflow/metrics.db", "File name for local metrics DB")
	metricsDBAggregationTimeInt := flag.Int(MetricsDBAggregationTime, 10, "Term time between aggregations")
	metricsDBClearTimeInt := flag.Int(MetricsDBClearTime, 600, "Metrics DB Clear Time")
	APIMetricsSendTimeInt := flag.Int(APIMetricsSendTime, 10, "APIMetric send term")
	collectorEnableOpenTelemetryB := flag.Bool(CollectorEnableOpenTelemetry, true, "Enable/Disable OpenTelemetry Collector")
	configDebugB := flag.Bool(Debug, false, "Enable/Disable debugging mode using logs")

	var flags []string
	flag.VisitAll(func(f *flag.Flag) {
		kv := fmt.Sprintf("%s:%v", f.Name, f.Value)
		flags = append(flags, kv)
	})
	log.Printf("Arguments [%s]", strings.Join(flags, " "))

	flag.Parse()

	viper.SetDefault(OtelGRPCListenAddr, *otelGRPCListenAddrStr)
	viper.SetDefault(OtelGRPCListenPort, *otelGRPCListenPortStr)
	viper.SetDefault(CustomExportListenAddr, *customExportListenAddrStr)
	viper.SetDefault(CustomExportListenPort, *customExportListenPortStr)
	viper.SetDefault(PatchNamespace, *patchNamespaceB)
	viper.SetDefault(PatchRestartDeployments, *patchRestartDeploymentsB)
	viper.SetDefault(AIEngineService, *aiEngineServiceStr)
	viper.SetDefault(AIEngineServicePort, *aiEngineServicePortStr)
	viper.SetDefault(AIEngineBatchSize, *aiEngineBatchSizeInt)
	viper.SetDefault(MetricsDBFileName, *metricsDBFileNameStr)
	viper.SetDefault(MetricsDBAggregationTime, *metricsDBAggregationTimeInt)
	viper.SetDefault(MetricsDBClearTime, *metricsDBClearTimeInt)
	viper.SetDefault(APIMetricsSendTime, *APIMetricsSendTimeInt)
	viper.SetDefault(CollectorEnableOpenTelemetry, *collectorEnableOpenTelemetryB)
	viper.SetDefault(Debug, *configDebugB)
}

// LoadConfig Load configuration
func LoadConfig() error {
	// Read configuration from command line
	readCmdLineParams()

	// Read environment variable, those are upper-cased
	viper.AutomaticEnv()

	// todo: read configuration from config file
	_ = os.Getenv("SENTRYFLOW_CFG")

	GlobalCfg.OtelGRPCListenAddr = viper.GetString(OtelGRPCListenAddr)
	GlobalCfg.OtelGRPCListenPort = viper.GetString(OtelGRPCListenPort)
	GlobalCfg.CustomExportListenAddr = viper.GetString(CustomExportListenAddr)
	GlobalCfg.CustomExportListenPort = viper.GetString(CustomExportListenPort)
	GlobalCfg.PatchNamespace = viper.GetBool(PatchNamespace)
	GlobalCfg.PatchRestartDeployments = viper.GetBool(PatchRestartDeployments)
	GlobalCfg.AIEngineService = viper.GetString(AIEngineService)
	GlobalCfg.AIEngineServicePort = viper.GetString(AIEngineServicePort)
	GlobalCfg.AIEngineBatchSize = viper.GetInt(AIEngineBatchSize)
	GlobalCfg.MetricsDBFileName = viper.GetString(MetricsDBFileName)
	GlobalCfg.MetricsDBAggregationTime = viper.GetInt(MetricsDBAggregationTime)
	GlobalCfg.MetricsDBClearTime = viper.GetInt(MetricsDBClearTime)
	GlobalCfg.APIMetricsSendTime = viper.GetInt(APIMetricsSendTime)
	GlobalCfg.CollectorEnableOpenTelemetry = viper.GetBool(CollectorEnableOpenTelemetry)
	GlobalCfg.Debug = viper.GetBool(Debug)

	log.Printf("Configuration [%+v]", GlobalCfg)

	return nil
}
