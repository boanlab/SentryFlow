package common

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr string
	ListenPort int
	ToExport   []string
}

// Cfg is for global reference
var Cfg Config

// LoadEnvVars loads environment variables and stores them as global variable
func LoadEnvVars() (Config, error) {
	var err error

	// load listen address and check if valid
	Cfg.ListenAddr = os.Getenv("LISTEN_ADDR")
	ip := net.ParseIP(Cfg.ListenAddr)
	if ip == nil {
		msg := fmt.Sprintf("invalid listen address %s", Cfg.ListenAddr)
		return Cfg, errors.New(msg)
	}
	Cfg.ListenAddr = ip.String()

	// load listen port and check if valid
	Cfg.ListenPort, err = strconv.Atoi(os.Getenv("LISTEN_PORT"))
	if err != nil {
		msg := fmt.Sprintf("invalid listen port %s: %v", os.Getenv("LISTEN_PORT"), err)
		return Cfg, errors.New(msg)
	}

	// Parse targets to export our gRPC format
	Cfg.ToExport, err = parseToExport()
	if err != nil {
		return Cfg, err
	}

	return Cfg, nil
}

// parseToExport parses the environment variable TO_EXPORT for exporting our data into remote servers
// The format of TO_EXPORT must be separated with a comma for each server.
// ex) TO_EXPORT=192.168.0.1:8080,myservice:9080
func parseToExport() ([]string, error) {
	ret := make([]string, 0)
	raw := os.Getenv("TO_EXPORT")

	// Meant that there was no such place to export our access logs
	if raw == "" {
		return ret, nil
	}

	// If not, start parsing with comma, but preprocess
	raw = strings.ReplaceAll(raw, " ", "")
	split := strings.Split(raw, ",")
	for _, server := range split {
		// Just check if the format was something like service:port format
		// Will NOT check exceptional cases here. This is just user's fault.
		if strings.Contains(server, ":") {
			ret = append(ret, server)
		} else {
			msg := fmt.Sprintf("invalid export server format %s", server)
			return ret, errors.New(msg)
		}
	}

	return ret, nil
}
