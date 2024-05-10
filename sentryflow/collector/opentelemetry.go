// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"strconv"
	"strings"

	"github.com/5gsec/SentryFlow/k8s"
	"github.com/5gsec/SentryFlow/processor"
	"github.com/5gsec/SentryFlow/protobuf"
	"github.com/5gsec/SentryFlow/types"
	otelLogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
)

// == //

// OpenTelemetryLogsServer structure
type OpenTelemetryLogsServer struct {
	otelLogs.UnimplementedLogsServiceServer
	collectorInterface
}

// newOpenTelemetryLogsServer Function
func newOpenTelemetryLogsServer() *OpenTelemetryLogsServer {
	ret := &OpenTelemetryLogsServer{}
	return ret
}

// registerService Function
func (otlLogs *OpenTelemetryLogsServer) registerService(server *grpc.Server) {
	otelLogs.RegisterLogsServiceServer(server, otlLogs)
}

// == //

// generateAPILogsFromOtel Function
func generateAPILogsFromOtel(logText string) []*protobuf.APILog {
	apiLogs := make([]*protobuf.APILog, 0)

	// Preprocess redundant chars
	logText = strings.ReplaceAll(logText, `\"`, "")
	logText = strings.ReplaceAll(logText, `}`, "")

	// Split logs by log_records, this is a single access log instance
	parts := strings.Split(logText, "log_records")
	if len(parts) == 0 {
		return nil
	}

	// Ignore the first entry (the metadata "resource_logs:{resource:{ scope_logs:{" part)
	for _, accessLog := range parts[0:] {
		var srcIP string
		var srcPort string
		var dstIP string
		var dstPort string

		if len(accessLog) == 0 {
			continue
		}

		index := strings.Index(accessLog, "string_value:\"")
		if index == -1 {
			continue
		}

		words := strings.Fields(accessLog[index+len("string_value:\""):])

		timeStamp := words[0]
		method := words[1]
		path := words[2]
		protocol := words[3]
		resCode, _ := strconv.ParseInt(words[4], 10, 64)

		srcInform := words[21]

		// Extract the left and right words based on the colon delimiter (ADDR:PORT)
		colonIndex := strings.LastIndex(srcInform, ":")
		if colonIndex > 0 && colonIndex < len(srcInform)-1 {
			srcIP = strings.TrimSpace(srcInform[:colonIndex])
			srcPort = strings.TrimSpace(srcInform[colonIndex+1:])
		}
		src := k8s.LookupK8sResource(srcIP)

		dstInform := words[20]

		// Extract the left and right words based on the colon delimiter (ADDR:PORT)
		colonIndex = strings.LastIndex(dstInform, ":")
		if colonIndex > 0 && colonIndex < len(dstInform)-1 {
			dstIP = strings.TrimSpace(dstInform[:colonIndex])
			dstPort = strings.TrimSpace(dstInform[colonIndex+1:])
		}
		dst := k8s.LookupK8sResource(dstIP)

		// Create APILog
		apiLog := protobuf.APILog{
			Id:        0, // @todo zero for now
			TimeStamp: timeStamp,

			SrcNamespace: src.Namespace,
			SrcName:      src.Name,
			SrcLabel:     src.Labels,
			SrcIP:        srcIP,
			SrcPort:      srcPort,
			SrcType:      types.K8sResourceTypeToString(src.Type),

			DstNamespace: dst.Namespace,
			DstName:      dst.Name,
			DstLabel:     dst.Labels,
			DstIP:        dstIP,
			DstPort:      dstPort,
			DstType:      types.K8sResourceTypeToString(dst.Type),

			Protocol:     protocol,
			Method:       method,
			Path:         path,
			ResponseCode: int32(resCode),
		}

		apiLogs = append(apiLogs, &apiLog)
	}

	return apiLogs
}

// Export Function for Log.Export in OpenTelemetry format
func (otlLogs *OpenTelemetryLogsServer) Export(_ context.Context, req *otelLogs.ExportLogsServiceRequest) (*otelLogs.ExportLogsServiceResponse, error) {
	apiLogs := generateAPILogsFromOtel(req.String())
	for _, apiLog := range apiLogs {
		processor.InsertAPILog(apiLog)
	}

	// @todo not consider partial success
	ret := otelLogs.ExportLogsServiceResponse{
		PartialSuccess: nil,
	}

	return &ret, nil
}

// == //
