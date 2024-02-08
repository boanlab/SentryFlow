package otel

import (
	"context"
	logs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"numbat/core"
	"numbat/log"
)

// olh Package level reference for OTEL log server
var olh *OtelLogServer

// init Function
func init() {
	olh = NewOtelLogServer()
}

// OtelLogServer structure
type OtelLogServer struct {
	logs.UnimplementedLogsServiceServer
}

// NewOtelLogServer Function
func NewOtelLogServer() *OtelLogServer {
	return new(OtelLogServer)
}

// Export Function
func (ols *OtelLogServer) Export(_ context.Context, req *logs.ExportLogsServiceRequest) (*logs.ExportLogsServiceResponse, error) {
	// This is for Log.Export in OpenTelemetry format
	als := log.GenerateLogs(req.String())

	for _, al := range als {
		core.Lh.InsertLog(al)
	}

	// For now, we will not consider partial success
	ret := logs.ExportLogsServiceResponse{
		PartialSuccess: nil,
	}

	return &ret, nil
}
