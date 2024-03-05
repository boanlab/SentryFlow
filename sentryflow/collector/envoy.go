package collector

import (
	envoyAls "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
	envoyMetrics "github.com/envoyproxy/go-control-plane/envoy/service/metrics/v3"
	"google.golang.org/grpc"
	"io"
	"log"
)

// EnvoyMetricsServer Structure
type EnvoyMetricsServer struct {
	envoyMetrics.UnimplementedMetricsServiceServer
	collectorInterface
}

// newEnvoyMetricsServer Function
func newEnvoyMetricsServer() *EnvoyMetricsServer {
	ret := &EnvoyMetricsServer{}
	return ret
}

// registerService Function
func (ems *EnvoyMetricsServer) registerService(server *grpc.Server) {
	envoyMetrics.RegisterMetricsServiceServer(server, ems)
}

// StreamMetrics Function
func (ems *EnvoyMetricsServer) StreamMetrics(stream envoyMetrics.MetricsService_StreamMetricsServer) error {
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Printf("[Envoy] Something went on wrong when receiving event: %v", err)
			return err
		}

		err = event.ValidateAll()
		if err != nil {
			log.Printf("[Envoy] Failed to validate stream: %v", err)
		}

		// @todo parse this event entry into our format
		identifier := event.GetIdentifier()
		identifier.GetNode().GetMetadata()

		if identifier != nil {
			log.Printf("ID: %s, %s", identifier.GetNode().GetId(), identifier.GetNode().GetCluster())
			log.Printf("Metrics:")
			for _, metric := range event.GetEnvoyMetrics() {
				log.Printf(" - %s(%s): %s", metric.GetName(), metric.GetType(), metric.GetMetric())
			}
		}

	}
}

// EnvoyAccessLogsServer Structure
type EnvoyAccessLogsServer struct {
	envoyAls.UnimplementedAccessLogServiceServer
	collectorInterface
}

// newEnvoyAccessLogsServer Function
func newEnvoyAccessLogsServer() *EnvoyAccessLogsServer {
	ret := &EnvoyAccessLogsServer{}
	return ret
}

// registerService Function
func (eas *EnvoyAccessLogsServer) registerService(server *grpc.Server) {
	envoyAls.RegisterAccessLogServiceServer(server, eas)
}

// StreamAccessLogs Function
func (eas *EnvoyAccessLogsServer) StreamAccessLogs(stream envoyAls.AccessLogService_StreamAccessLogsServer) error {
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			log.Printf("[Envoy] Something went on wrong when receiving event: %v", err)
			return err
		}

		err = event.ValidateAll()
		if err != nil {
			log.Printf("[Envoy] Failed to validate stream: %v", err)
		}

		// Check HTTP logs first, then TCP Logs
		// Envoy will send HTTP logs with higher priority.
		if event.GetHttpLogs() != nil {
			for _, entry := range event.GetHttpLogs().LogEntry {
				identifier := event.GetIdentifier()
				if identifier != nil {
					log.Printf("=====================[ACCESS LOG - HTTP]=====================")
					log.Printf("ID: %s, %s", identifier.GetNode().GetId(), identifier.GetNode().GetCluster())

					// @todo parse this entry into proto.Log format
					req := entry.GetRequest()
					resp := entry.GetResponse()
					proto := entry.GetProtocolVersion()
					comm := entry.GetCommonProperties()
					log.Printf("Request: request=%v, resp=%v, proto=%v, comm=%v",
						req.String(), resp.String(), proto.String(), comm.String())
				}
			}
		}

		// Check TCP logs later
		// In Envoy, even if HTTP is based on TCP, it will just send HTTP logs
		if event.GetTcpLogs() != nil {
			for _, entry := range event.GetTcpLogs().LogEntry {
				identifier := event.GetIdentifier()
				// @todo parse this entry into proto.Log format

				if identifier != nil {
					log.Printf("=====================[ACCESS LOG - TCP]=====================")
					log.Printf("ID: %s, %s", identifier.GetNode().GetId(), identifier.GetNode().GetCluster())
					log.Printf("Data: %v", entry.String())
				}
			}
		}
	}
}
