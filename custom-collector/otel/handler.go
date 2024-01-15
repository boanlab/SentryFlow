package otel

import (
	"context"
	"custom-collector/protobuf"
	"fmt"
	logs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	"log"
	"net"
	"strconv"
	"strings"
)

// Handler is the handler for OTEL collector
type Handler struct {
	addr      string
	lis       net.Listener
	server    *grpc.Server
	logServer LogServer
}

// NewHandler creates a handler for OTEL collector
func NewHandler(addr string) *Handler {
	// Create a handler object
	h := Handler{}
	var err error

	// Dump address
	h.addr = addr

	// Start listening 4317
	h.lis, err = net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Could not start up server at %s: %v", h.lis, err)
	}

	// Create a new gRPC server
	h.server = grpc.NewServer()

	// Register log collector server for Logging
	h.logServer = LogServer{}
	logs.RegisterLogsServiceServer(h.server, h.logServer)

	return &h
}

// Serve starts the gRPC server for OTEL collection
func (h *Handler) Serve() error {
	log.Printf("[OTEL] Started to serve on %s", h.addr)
	err := h.server.Serve(h.lis)
	if err != nil {
		log.Fatalf("[OTEL] Could not start up gRPC server at %s: %v", h.addr, err)
		return err
	}

}

// LogServer is for exporting log handler
type LogServer struct {
	logs.UnimplementedLogsServiceServer
}

// Export interface function for LogServiceServer
func (ls LogServer) Export(c context.Context, request *logs.ExportLogsServiceRequest) (*logs.ExportLogsServiceResponse, error) {
	var logText string
	var index int

	logText = request.String()
	logText = strings.ReplaceAll(logText, `\"`, "")
	logText = strings.ReplaceAll(logText, `}`, "")
	parts := strings.Split(logText, "\\n\"")

	for _, part := range parts[0:] {
		if len(part) <= 0 {
			continue
		}
		index = strings.Index(part, "string_value:\"")
		if index == -1 {
			continue
		}
		result := part[index+len("string_value:\""):]
		words := strings.Fields(result)

		method := words[1]
		path := words[2]
		protocolName := words[3]
		timeStamp := words[0]
		resCode, _ := strconv.ParseInt(words[4], 10, 64)

		srcInform := words[21]
		dstInform := words[20]

		var srcIP string
		var dstIP string
		var srcPort string
		var dstPort string

		var colonIndex int
		// Extract the left and right words based on the colon delimiter (ADDR:PORT)
		colonIndex = strings.LastIndex(srcInform, ":")
		if colonIndex > 0 && colonIndex < len(srcInform)-1 {
			srcIP = strings.TrimSpace(srcInform[:colonIndex])
			srcPort = strings.TrimSpace(srcInform[colonIndex+1:])
		}
		colonIndex = strings.LastIndex(dstInform, ":")
		if colonIndex > 0 && colonIndex < len(dstInform)-1 {
			dstIP = strings.TrimSpace(dstInform[:colonIndex])
			dstPort = strings.TrimSpace(dstInform[colonIndex+1:])
		}

		var srcName string
		var srcNs string
		var srcLabel map[string]string
		var dstName string
		var dstNs string
		var dstLabel map[string]string

		// Get source IP addr, port
		if pod, ok := podsMap[srcIP]; ok {
			srcName = pod.Name
			srcNs = pod.Namespace
			srcLabel = pod.Labels
		} else if service, ok := servicesMap[srcIP]; ok {
			srcName = service.Name
			srcNs = service.Namespace
			srcLabel = service.Labels
		} else {
			srcName = "Not found"
			srcNs = "Not found"
		}

		// Get destination IP addr, port
		if pod, ok := podsMap[dstIP]; ok {
			dstName = pod.Name
			dstNs = pod.Namespace
			dstLabel = pod.Labels
		} else if service, ok := servicesMap[dstIP]; ok {
			dstName = service.Name
			dstNs = service.Namespace
			dstLabel = service.Labels
		} else {
			dstName = "Not found"
			dstNs = "Not found"
		}

		// Create a result log struct - MyLog
		myLog := protobuf.MyLog{
			TimeStamp:        timeStamp,
			Id:               id + strconv.Itoa(num),
			SrcNamespaceName: srcNs,
			SrcPodName:       srcName,
			SrcLabel:         srcLabel,
			SrcIP:            srcIP,
			SrcPort:          srcPort,
			DstNamespaceName: dstNs,
			DstPodName:       dstName,
			DstLabel:         dstLabel,
			DstIP:            dstIP,
			DstPort:          dstPort,
			Protocol:         protocolName,
			Method:           method,
			Path:             path,
			ResponseCode:     resCode,
		}

		num++

		// Ignore non HTTP protocols
		if strings.Contains(myLog.Protocol, "HTTP") {
			log.Printf("%+v\n", myLog)
		}

		// 서버로부터 받은 응답 처리
		fmt.Printf("Response from server: %+v\n", response)
	}

	ret := logs.ExportLogsServiceResponse{PartialSuccess: nil}
	return &ret, nil
}
