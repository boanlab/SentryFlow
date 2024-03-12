// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"github.com/5GSEC/sentryflow/core"
	otelLogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
)

// OtelLogServer structure
type OtelLogServer struct {
	otelLogs.UnimplementedLogsServiceServer
	collectorInterface
}

// newOtelLogServer Function
func newOtelLogServer() *OtelLogServer {
	ret := &OtelLogServer{}
	return ret
}

// registerService Function
func (ols *OtelLogServer) registerService(server *grpc.Server) {
	otelLogs.RegisterLogsServiceServer(server, ols)
}

// Export Function
func (ols *OtelLogServer) Export(_ context.Context, req *otelLogs.ExportLogsServiceRequest) (*otelLogs.ExportLogsServiceResponse, error) {
	// This is for Log.Export in OpenTelemetry format
	als := core.GenerateAccessLogsFromOtel(req.String())

	for _, al := range als {
		core.Lh.InsertLog(al)
	}

	// For now, we will not consider partial success
	ret := otelLogs.ExportLogsServiceResponse{
		PartialSuccess: nil,
	}

	return &ret, nil
}
