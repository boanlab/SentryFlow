package exporter

import (
	"errors"
	"fmt"
	"github.com/emicklei/go-restful/v3/log"
	"google.golang.org/grpc"
	"net"
	cfg "numbat/config"
	"numbat/protobuf"
	"sync"
	"time"
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

	listener   net.Listener
	gRPCServer *grpc.Server
}

// NewExporterHandler Function
func NewExporterHandler() *ExporterHandler {
	exp := &ExporterHandler{
		baseExecutionID: uint64(time.Now().UnixMicro()),
		currentLogCount: 0,
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

	Exp.logChannel <- al
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
