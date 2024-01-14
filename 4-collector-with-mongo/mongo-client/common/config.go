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

var cfg Config

// LoadEnvVars loads environment variables and stores them as global variable
func LoadEnvVars() (Config, error) {
	var err error

	// load listen address and check if valid
	cfg.ListenAddr = os.Getenv("LISTEN_ADDR")
	ip := net.ParseIP(cfg.ListenAddr)
	if ip == nil {
		msg := fmt.Sprintf("invalid listen address %s", cfg.ListenAddr)
		return cfg, errors.New(msg)
	}
	cfg.ListenAddr = ip.String()

	// load listen port and check if valid
	cfg.ListenPort, err = strconv.Atoi(os.Getenv("LISTEN_PORT"))
	if err != nil {
		msg := fmt.Sprintf("invalid listen port %s: %v", os.Getenv("LISTEN_PORT"), err)
		return cfg, errors.New(msg)
	}

	return cfg, nil
}
