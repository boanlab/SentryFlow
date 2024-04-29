// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config Structure
type Config struct {
	Hostname string

	ServerAddr string
	ServerPort int

	LogCfg       string
	MetricCfg    string
	MetricFilter string
}

// Cfg for Global Reference
var Cfg Config

// LoadEnvVars loads environment variables and stores them in Cfg (global variables)
func LoadEnvVars() (Config, error) {
	var err error

	// get hostname
	Cfg.Hostname, err = os.Hostname()
	if err != nil {
		msg := fmt.Sprintf("[Config] Could not find hostname: %v", err)
		return Cfg, errors.New(msg)
	}

	// load listen address and check if valid
	Cfg.ServerAddr = os.Getenv("SERVER_ADDR")

	// load listen port and check if valid
	Cfg.ServerPort, err = strconv.Atoi(os.Getenv("SERVER_PORT"))
	if err != nil {
		msg := fmt.Sprintf("[Config] Invalid server port %s: %v", os.Getenv("SERVER_PORT"), err)
		return Cfg, errors.New(msg)
	}

	Cfg.LogCfg = os.Getenv("LOG_CFG")
	Cfg.MetricCfg = os.Getenv("METRIC_CFG")
	Cfg.MetricFilter = os.Getenv("METRIC_FILTER")

	return Cfg, nil
}
