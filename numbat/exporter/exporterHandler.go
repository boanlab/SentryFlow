// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"errors"
	"fmt"
	"net"
	cfg "numbat/config"
	"numbat/protobuf"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3/log"
	"google.golang.org/grpc"
)

// Exp global reference for Exporter Handler
var Exp *ExporterHandler

// init Function
func init() {
	Exp = NewExporterHandler()
}

// ExporterHandler structure
type ExporterHandler struct {
	baseExecutionID uint64
	currentLogCount uint64
	logChannel      chan *protobuf.Log
	lock            sync.Mutex // @todo find better solution for this
	stopChan        chan struct{}

	exporters []*ExporterInform

	listener   net.Listener
	gRPCServer *grpc.Server
}

// ExproterInform structure
type ExporterInform struct {
	stream    protobuf.Logs_SendServer
	Hostname  string
	IpAddress string
}

// NewExporterHandler Function
func NewExporterHandler() *ExporterHandler {
	exp := &ExporterHandler{
		baseExecutionID: uint64(time.Now().UnixMicro()),
		currentLogCount: 0,
		exporters:       make([]*ExporterInform, 0),
		logChannel:      make(chan *protobuf.Log),
		stopChan:        make(chan struct{}),
		lock:            sync.Mutex{},
	}

	return exp
}

// InsertAccessLog Function
func InsertAccessLog(al *protobuf.Log) {
	// Avoid race condition for currentLogCount, otherwise we might have duplicate IDs
	Exp.lock.Lock()
	al.Id = Exp.baseExecutionID + Exp.currentLogCount
	Exp.currentLogCount++
	Exp.lock.Unlock()

	// Send stream with replies
	// @todo: make max failure count for a single client
	for _, exp := range Exp.exporters {
		curRetry := 0
		for curRetry < 3 { // @todo make this retry count configurable using configs
			err := exp.stream.Send(al)
			if err != nil {
				log.Printf("[Error] Unable to send access log to %s(%s) (retry=%d/%d): %v",
					exp.Hostname, exp.IpAddress, curRetry, 3, err)
				curRetry++
			} else {
				break
			}
		}
	}
}

// InitExporterServer Function
func (exp *ExporterHandler) InitExporterServer() error {
	listenAddr := fmt.Sprintf("%s:%s", cfg.GlobalCfg.CustomExportListenAddr, cfg.GlobalCfg.CustomExportListenPort)

	// Start listening
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		msg := fmt.Sprintf("unable to listen at %s: %v", listenAddr, err)
		return errors.New(msg)
	}

	// Create gRPC server
	server := grpc.NewServer()
	protobuf.RegisterNumbatServer(server, exs)

	exp.listener = lis
	exp.gRPCServer = server

	log.Printf("[Exporter] Exporter listening at %s", listenAddr)
	return nil
}

// StartExporterServer Function
func (exp *ExporterHandler) StartExporterServer(wg *sync.WaitGroup) error {
	log.Printf("[Exporter] Starting exporter server")
	var err error
	err = nil

	go func() {
		wg.Add(1)
		// Serve is blocking function
		err = exp.gRPCServer.Serve(exp.listener)
		if err != nil {
			wg.Done()
			return
		}

		wg.Done()
	}()

	return err
}

// StopExporterServer Function
func (exp *ExporterHandler) StopExporterServer() {
	// Gracefully stop all client connections
	exp.stopChan <- struct{}{}

	// Gracefully stop gRPC Server
	exp.gRPCServer.GracefulStop()

	log.Printf("[Exporter] Stopped exporter server")
}
