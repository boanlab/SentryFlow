package exporter

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"numbat/common"
	protobuf "numbat/protobuf"
	"os"
	"time"
)

// Handler manages the exporting process using our own protocol
type Handler struct {
	executionID int64 // For unique timestamp for current execution
	curIndex    int   // The index of current entry from the beginning
	exporters   map[string]protobuf.LogsClient
	channel     chan *protobuf.Log
}

// Manager is for global reference
var Manager *Handler

// NewHandler creates the handler for exporter
func NewHandler() *Handler {
	h := &Handler{
		executionID: time.Now().UnixMicro(), // We do not need nano precision, just micro is okay for this
		curIndex:    0,
		exporters:   make(map[string]protobuf.LogsClient),
		channel:     make(chan *protobuf.Log),
	}

	// Try connecting exporters
	h.tryExporters()

	// Everything was successful
	Manager = h
	return h
}

// RunExporters start running goroutines for exportRoutine with given channel as stop
func (h *Handler) RunExporters(stopCh chan os.Signal) {
	go func() {
		h.exportRoutine(stopCh)
	}()
}

// exportRoutine exports the Access Logs in the channel to all available exporter
// This will not retry failed exports for now
// @todo add a simple retry for exporting
func (h *Handler) exportRoutine(stopCh chan os.Signal) {
	for {
		select {
		// This was from access log channel
		case accessLog, ok := <-h.channel:
			// Channel closed, exit the goroutine
			if !ok {
				return
			}

			// If not, iterate and export all to exporters
			for name, exporter := range h.exporters {
				accessLog.Id = uint64(h.executionID) + uint64(h.curIndex)
				_, err := exporter.Send(context.Background(), accessLog)
				if err != nil {
					log.Printf("[EXPORTER] Failed exporting log to exporter %s: %v", name, err)
				}

				h.curIndex = h.curIndex + 1
			}

		// This was from stopCh which kills the program
		case <-stopCh:
			return
		}
	}
}

// InsertAccessLog inserts a new AccessLog message into the exporter channel
// This will act as a buffer to stop race conditions
func (h *Handler) InsertAccessLog(al *protobuf.Log) {
	h.channel <- al
}

// tryExporters will connect all exporters listed in the config,
// This will ignore the ones which failed
func (h *Handler) tryExporters() {
	log.Printf("[EXPORTER] Connecting exporters...")
	for _, addr := range common.Cfg.ToExport {
		conn, err := grpc.Dial(addr, grpc.WithInsecure()) // @todo implement this as secure
		if err != nil {
			log.Printf("[EXPORTER] Could not dial gRPC to exporter %s: %v", addr, err)
		} else {
			client := protobuf.NewLogsClient(conn)
			log.Printf("[EXPORTER] Successfully connected exporter %s", addr)
			h.exporters[addr] = client
		}
	}
}
