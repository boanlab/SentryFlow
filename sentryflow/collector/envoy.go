// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"io"
	"log"
	"strconv"

	"github.com/5gsec/SentryFlow/k8s"
	"github.com/5gsec/SentryFlow/processor"
	"github.com/5gsec/SentryFlow/protobuf"
	"github.com/5gsec/SentryFlow/types"

	envoyAccLogsData "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v3"
	envoyAccLogs "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
	envoyMetrics "github.com/envoyproxy/go-control-plane/envoy/service/metrics/v3"

	"google.golang.org/grpc"
)

// == //

// EnvoyAccessLogsServer Structure
type EnvoyAccessLogsServer struct {
	envoyAccLogs.UnimplementedAccessLogServiceServer
	collectorInterface
}

// newEnvoyAccessLogsServer Function
func newEnvoyAccessLogsServer() *EnvoyAccessLogsServer {
	ret := &EnvoyAccessLogsServer{}
	return ret
}

// registerService Function
func (evyAccLogs *EnvoyAccessLogsServer) registerService(server *grpc.Server) {
	envoyAccLogs.RegisterAccessLogServiceServer(server, evyAccLogs)
}

// == //

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
func (evyMetrics *EnvoyMetricsServer) registerService(server *grpc.Server) {
	envoyMetrics.RegisterMetricsServiceServer(server, evyMetrics)
}

// == //

// generateAPILogsFromEnvoy Function
func generateAPILogsFromEnvoy(entry *envoyAccLogsData.HTTPAccessLogEntry) *protobuf.APILog {
	comm := entry.GetCommonProperties()
	timeStamp := comm.GetStartTime().Seconds

	srcInform := entry.GetCommonProperties().GetDownstreamRemoteAddress().GetSocketAddress()
	srcIP := srcInform.GetAddress()
	srcPort := strconv.Itoa(int(srcInform.GetPortValue()))
	src := k8s.LookupK8sResource(srcIP)

	dstInform := entry.GetCommonProperties().GetUpstreamRemoteAddress().GetSocketAddress()
	dstIP := dstInform.GetAddress()
	dstPort := strconv.Itoa(int(dstInform.GetPortValue()))
	dst := k8s.LookupK8sResource(dstIP)

	request := entry.GetRequest()
	response := entry.GetResponse()

	protocol := entry.GetProtocolVersion().String()
	method := request.GetRequestMethod().String()
	path := request.GetPath()
	resCode := response.GetResponseCode().GetValue()

	envoyAPILog := &protobuf.APILog{
		Id:        0, // @todo zero for now
		TimeStamp: strconv.FormatInt(timeStamp, 10),

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

	return envoyAPILog
}

// StreamAccessLogs Function
func (evyAccLogs *EnvoyAccessLogsServer) StreamAccessLogs(stream envoyAccLogs.AccessLogService_StreamAccessLogsServer) error {
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			log.Printf("[EnvoyAPILogs] Failed to receive an event: %v", err)
			return err
		}

		if event.GetHttpLogs() != nil {
			for _, entry := range event.GetHttpLogs().LogEntry {
				envoyAPILog := generateAPILogsFromEnvoy(entry)
				processor.InsertAPILog(envoyAPILog)
			}
		}
	}
}

// == //

// generateMetricsFromEnvoy Function
func generateMetricsFromEnvoy(event *envoyMetrics.StreamMetricsMessage, metaData map[string]interface{}) *protobuf.EnvoyMetrics {
	envoyMetrics := &protobuf.EnvoyMetrics{
		TimeStamp: "",

		Namespace: metaData["NAMESPACE"].(string),
		Name:      metaData["NAME"].(string),
		IPAddress: metaData["INSTANCE_IPS"].(string),
		Labels:    k8s.LookupK8sResource(metaData["INSTANCE_IPS"].(string)).Labels,

		Metrics: make(map[string]*protobuf.MetricValue),
	}

	envoyMetrics.Metrics["GAUGE"] = &protobuf.MetricValue{
		Value: make(map[string]string),
	}

	envoyMetrics.Metrics["COUNTER"] = &protobuf.MetricValue{
		Value: make(map[string]string),
	}

	envoyMetrics.Metrics["HISTOGRAM"] = &protobuf.MetricValue{
		Value: make(map[string]string),
	}

	envoyMetrics.Metrics["SUMMARY"] = &protobuf.MetricValue{
		Value: make(map[string]string),
	}

	for _, metric := range event.GetEnvoyMetrics() {
		metricType := metric.GetType().String()
		metricName := metric.GetName()

		if envoyMetrics.Metrics[metricType].Value == nil {
			continue
		}

		for _, metricDetail := range metric.GetMetric() {
			var metricValue string

			if envoyMetrics.TimeStamp == "" {
				envoyMetrics.TimeStamp = strconv.FormatInt(metricDetail.GetTimestampMs(), 10)
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

			envoyMetrics.Metrics[metricType].Value[metricName] = metricValue
		}
	}

	return envoyMetrics
}

// StreamMetrics Function
func (evyMetrics *EnvoyMetricsServer) StreamMetrics(stream envoyMetrics.MetricsService_StreamMetricsServer) error {
	event, err := stream.Recv()
	if err == io.EOF {
		return nil
	} else if err != nil {
		log.Printf("[EnvoyMetrics] Failed to receive an event: %v", err)
		return err
	}

	err = event.ValidateAll()
	if err != nil {
		log.Printf("[EnvoyMetrics] Failed to validate an event: %v", err)
	}

	identifier := event.GetIdentifier()
	if identifier != nil {
		metaData := identifier.GetNode().GetMetadata().AsMap()
		envoyMetrics := generateMetricsFromEnvoy(event, metaData)
		processor.InsertMetrics(envoyMetrics)
	}

	return nil
}

// == //
