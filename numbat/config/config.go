package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// NumbatConfig structure
type NumbatConfig struct {
	OtelGRPCListenAddr string // IP address to use for OTEL gRPC
	OtelGRPCListenPort string // Port to use for OTEL gRPC

	CustomExportListenAddr string // IP address to use for custom exporter gRPC
	CustomExportListenPort string // Port to use for custom exporter gRPC

	Debug bool // Enable/Disable Numbat debug mode
}

// GlobalCfg Global configuration for Numbat
var GlobalCfg NumbatConfig

// Config const
const (
	OtelGRPCListenAddr     string = "otelGRPCListenAddr"
	OtelGRPCListenPort     string = "otelGRPCListenPort"
	CustomExportListenAddr string = "customExportListenAddr"
	CustomExportListenPort string = "customExportListenPort"
	Debug                  string = "debug"
)

func readCmdLineParams() {
	otelGRPCListenAddrStr := flag.String(OtelGRPCListenAddr, "0.0.0.0", "OTEL gRPC server listen address")
	otelGRPCListenPortStr := flag.String(OtelGRPCListenPort, "4317", "OTEL gRPC server listen port")
	customExportListenAddrStr := flag.String(CustomExportListenAddr, "0.0.0.0", "Custom export gRPC server listen address")
	customExportListenPortStr := flag.String(CustomExportListenPort, "8080", "Custom export gRPC server listen port")
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
	viper.SetDefault(Debug, *configDebugB)
}

// LoadConfig Load configuration
func LoadConfig() error {
	// Read configuration from command line
	readCmdLineParams()

	// Read environment variable, those are upper-cased
	viper.AutomaticEnv()

	// todo: read configuration from config file
	_ = os.Getenv("NUMBAT_CFG")

	GlobalCfg.OtelGRPCListenAddr = viper.GetString(OtelGRPCListenAddr)
	GlobalCfg.OtelGRPCListenPort = viper.GetString(OtelGRPCListenPort)
	GlobalCfg.CustomExportListenAddr = viper.GetString(CustomExportListenAddr)
	GlobalCfg.CustomExportListenPort = viper.GetString(CustomExportListenPort)
	GlobalCfg.Debug = viper.GetBool(Debug)

	log.Printf("Configuration [%+v]", GlobalCfg)

	return nil
}
