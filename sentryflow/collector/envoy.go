// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"io"
	"log"
	"strconv"

	"github.com/5GSEC/SentryFlow/core"
	"github.com/5GSEC/SentryFlow/protobuf"
	envoyAls "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
	envoyMetrics "github.com/envoyproxy/go-control-plane/envoy/service/metrics/v3"
	"google.golang.org/grpc"
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
		metaData := identifier.GetNode().GetMetadata().AsMap()

		envoyMetric := &protobuf.EnvoyMetric{
			PodContainer: metaData["APP_CONTAINERS"].(string),
			PodIP:        metaData["INSTANCE_IPS"].(string),
			PodName:      metaData["NAME"].(string),
			PodNamespace: metaData["NAMESPACE"].(string),
			TimeStamp:    "",
			Metric: map[string]*protobuf.Metric{
				"GAUGE":     {MetricValue: []*protobuf.MetricValue{}},
				"COUNTER":   {MetricValue: []*protobuf.MetricValue{}},
				"HISTOGRAM": {MetricValue: []*protobuf.MetricValue{}},
				"SUMMARY":   {MetricValue: []*protobuf.MetricValue{}},
				"UNTYPED":   {MetricValue: []*protobuf.MetricValue{}},
				"LABEL":     {MetricValue: []*protobuf.MetricValue{}},
			},
		}

		for _, metric := range event.GetEnvoyMetrics() {
			metricType := metric.GetType().String()
			metricName := metric.GetName()

			if envoyMetric.Metric[metricType].MetricValue == nil {
				continue
			}

			var metricValue string

			for _, metricDetail := range metric.GetMetric() {
				if envoyMetric.TimeStamp == "" {
					envoyMetric.TimeStamp = strconv.FormatInt(metricDetail.GetTimestampMs(), 10)
				}
				if metricType == "GAUGE" {
					metricValue = strconv.FormatFloat(metricDetail.GetGauge().GetValue(), 'f', -1, 64)
				}
				if metricType == "COUNTER" {
					metricValue = strconv.FormatFloat(metricDetail.GetCounter().GetValue(), 'f', -1, 64)
				}
				if metricType == "HISTOGRAM" {
					metricValue = strconv.FormatUint(metricDetail.GetHistogram().GetSampleCount(), 10)
				}
				if metricType == "SUMMARY" {
					metricValue = strconv.FormatUint(metricDetail.GetHistogram().GetSampleCount(), 10)
				}

				curMetric := &protobuf.MetricValue{
					Name:  metricName,
					Value: metricValue,
				}

				envoyMetric.Metric[metricType].MetricValue = append(envoyMetric.Metric[metricType].MetricValue, curMetric)
			}
		}

		core.Lh.InsertLog(envoyMetric)
	}

	return nil
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

		// Check HTTP logs
		if event.GetHttpLogs() != nil {
			for _, entry := range event.GetHttpLogs().LogEntry {
				envoyAccessLog := core.GenerateAccessLogsFromEnvoy(entry)
				core.Lh.InsertLog(envoyAccessLog)
			}
		}
	}
}
