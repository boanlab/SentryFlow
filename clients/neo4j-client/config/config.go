// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config structure
type Config struct {
	Hostname string

	ServerAddr string
	ServerPort int

	NodeLevel string
	EdgeLevel string

	Neo4jURI      string
	Neo4jUsername string
	Neo4jPassword string
}

// Cfg is for global reference
var Cfg Config

// LoadEnvVars loads environment variables and stores them as global variable
func LoadEnvVars() (Config, error) {
	var err error

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
		msg := fmt.Sprintf("invalid server port %s: %v", os.Getenv("SERVER_PORT"), err)
		return Cfg, errors.New(msg)
	}

	Cfg.NodeLevel = os.Getenv("NODE_LEVEL")
	Cfg.EdgeLevel = os.Getenv("EDGE_LEVEL")

	Cfg.Neo4jURI = os.Getenv("NEO4J_URI")
	Cfg.Neo4jUsername = os.Getenv("NEO4J_USERNAME")
	Cfg.Neo4jPassword = os.Getenv("NEO4J_PASSWORD")

	return Cfg, nil
}
