package common

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
)

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
	ip := net.ParseIP(Cfg.ServerAddr)
	if ip == nil {
		msg := fmt.Sprintf("invalid server address %s", Cfg.ServerAddr)
		return Cfg, errors.New(msg)
	}
	Cfg.ServerAddr = ip.String()

	// load listen port and check if valid
	Cfg.ServerPort, err = strconv.Atoi(os.Getenv("SERVER_PORT"))
	if err != nil {
		msg := fmt.Sprintf("invalid server port %s: %v", os.Getenv("SERVER_PORT"), err)
		return Cfg, errors.New(msg)
	}

	return Cfg, nil
}
