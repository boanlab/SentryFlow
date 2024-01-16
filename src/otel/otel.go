package otel

import (
	"context"
	"custom-collector/exporter"
	logs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
)

// LogServer is for exporting log handler
type LogServer struct {
	logs.UnimplementedLogsServiceServer
}

// Export interface function for LogServiceServer
func (ls LogServer) Export(c context.Context, request *logs.ExportLogsServiceRequest) (*logs.ExportLogsServiceResponse, error) {
	// Convert logText into AccessLogs
	accessLogs := parseAccessLog(request.String())

	// Export the parsed access logs to exporter
	for _, al := range accessLogs {
		exporter.Manager.InsertAccessLog(al)
	}

	// Return fully successful log for the gRPC response
	ret := logs.ExportLogsServiceResponse{PartialSuccess: nil}
	return &ret, nil
}
