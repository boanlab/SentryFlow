// SPDX-License-Identifier: Apache-2.0

package common

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config structure
type Config struct {
	ServerAddr string
	ServerPort int
}

// Cfg is for global reference
var Cfg Config

// LoadEnvVars loads environment variables and stores them as global variable
func LoadEnvVars() (Config, error) {
	var err error

	// load listen address and check if valid
	Cfg.ServerAddr = os.Getenv("SERVER_ADDR")

	// load listen port and check if valid
	Cfg.ServerPort, err = strconv.Atoi(os.Getenv("SERVER_PORT"))
	if err != nil {
		msg := fmt.Sprintf("invalid server port %s: %v", os.Getenv("SERVER_PORT"), err)
		return Cfg, errors.New(msg)
	}

	return Cfg, nil
}
