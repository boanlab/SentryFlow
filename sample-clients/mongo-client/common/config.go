package common

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
)

type Config struct {
	ListenAddr string
	ListenPort int
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

	return Cfg, nil
}
