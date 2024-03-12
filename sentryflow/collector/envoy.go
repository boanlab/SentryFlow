// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"
	"github.com/5GSEC/sentryflow/core"
	"github.com/5GSEC/sentryflow/protobuf"
	"github.com/5GSEC/sentryflow/types"
	envoyAls "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
	envoyMetrics "github.com/envoyproxy/go-control-plane/envoy/service/metrics/v3"
	"google.golang.org/grpc"
	"io"
	"log"
	"strconv"
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
			log.Printf("[Envoy] Received EnvoyMetric - ID: %s, %s", identifier.GetNode().GetId(), identifier.GetNode().GetCluster())

			nodeID := identifier.GetNode().GetId()
			cluster := identifier.GetNode().GetCluster()

			curIdentifier := fmt.Sprintf("%s, %s", nodeID, cluster)
			envoyMetric := &protobuf.EnvoyMetric{
				Identifier: curIdentifier,
				Metric:     []*protobuf.Metric{},
			}

			for _, metric := range event.GetEnvoyMetrics() {
				metricType := metric.GetType().String()
				metricName := metric.GetName()
				tempMetrics := metric.GetMetric()
				metrics := fmt.Sprintf("%s", tempMetrics)

				curMetric := &protobuf.Metric{
					Type:  metricType,
					Key:   metricName,
					Value: metrics,
				}

				envoyMetric.Metric = append(envoyMetric.Metric, curMetric)
			}

			core.Lh.InsertLog(envoyMetric)
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
			log.Printf("[Envoy] Failed to validate event: %v", err)
			continue
		}

		// Check HTTP logs
		// Envoy will send HTTP logs with higher priority.
		if event.GetHttpLogs() != nil {
			for _, entry := range event.GetHttpLogs().LogEntry {
				identifier := event.GetIdentifier()
				if identifier != nil {
					log.Printf("[Envoy] Received EnvoyAccessLog - ID: %s, %s", identifier.GetNode().GetId(), identifier.GetNode().GetCluster())

					srcInform := entry.GetCommonProperties().GetDownstreamRemoteAddress().GetSocketAddress()
					srcIP := srcInform.GetAddress()
					srcPort := strconv.Itoa(int(srcInform.GetPortValue()))
					src := core.LookupNetworkedResource(srcIP)

					dstInform := entry.GetCommonProperties().GetUpstreamRemoteAddress().GetSocketAddress()
					dstIP := dstInform.GetAddress()
					dstPort := strconv.Itoa(int(dstInform.GetPortValue()))
					dst := core.LookupNetworkedResource(dstIP)

					req := entry.GetRequest()
					res := entry.GetResponse()
					comm := entry.GetCommonProperties()
					proto := entry.GetProtocolVersion()

					timeStamp := comm.GetStartTime().Seconds
					path := req.GetPath()
					method := req.GetRequestMethod().String()
					protocolName := proto.String()
					resCode := res.GetResponseCode().GetValue()

					envoyAccessLog := &protobuf.APILog{
						TimeStamp:    strconv.FormatInt(timeStamp, 10),
						Id:           0, //  do 0 for now, we are going to write it later
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
						Protocol:     protocolName,
						Method:       method,
						Path:         path,
						ResponseCode: int32(resCode),
					}

					core.Lh.InsertLog(envoyAccessLog)
				}
			}
		}
	}
}
