package log

import (
	corev1 "k8s.io/api/core/v1"
	"numbat/k8s"
	protobuf "numbat/protobuf"
	"strconv"
	"strings"
)

// parseAccessLog parses the string access log coming from OTEL
// @todo this needs more optimization, this code is kind of messy
func GenerateLog(logText string) []*protobuf.Log {
	// Create a array of AccessLogs for returning gRPC comm
	var index int
	ret := make([]*protobuf.Log, 0)

	// Preprocess redundant chars
	logText = strings.ReplaceAll(logText, `\"`, "")
	logText = strings.ReplaceAll(logText, `}`, "")

	// Split logs by log_records, this is single access log instance
	parts := strings.Split(logText, "log_records")
	if len(parts) == 0 {
		return nil
	}

	// Ignore the first entry, this was the metadata "resource_logs:{resource:{ scope_logs:{" part.
	for _, al := range parts[0:] {
		if len(al) == 0 {
			continue
		}

		index = strings.Index(al, "string_value:\"")
		if index == -1 {
			continue
		}

		result := al[index+len("string_value:\""):]
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

		// Find Kubernetes resource details from src and dst IP (service or a pod)
		srcName, srcNamespace, srcLabel, srcResourceType := findResourceDetails(srcIP)
		dstName, dstNamespace, dstLabel, dstResourceType := findResourceDetails(dstIP)

		// Create AccessLog in our gRPC format
		cur := protobuf.Log{
			TimeStamp:    timeStamp,
			Id:           0, //  do 0 for now, we are going to write it later
			SrcNamespace: srcNamespace,
			SrcName:      srcName,
			SrcLabel:     srcLabel,
			SrcIP:        srcIP,
			SrcPort:      srcPort,
			SrcType:      srcResourceType,
			DstNamespace: dstNamespace,
			DstName:      dstName,
			DstLabel:     dstLabel,
			DstIP:        dstIP,
			DstPort:      dstPort,
			DstType:      dstResourceType,
			Protocol:     protocolName,
			Method:       method,
			Path:         path,
			ResponseCode: resCode,
		}

		ret = append(ret, &cur)
	}

	return ret
}

// findResourceDetails returns name, namespace and the labels attached for this Kubernetes resource and its type
func findResourceDetails(srcIP string) (string, string, map[string]string, string) {
	name := "Not found"
	namespace := "Not found"
	labels := make(map[string]string)

	// Find Kubernetes resource from source IP (service or a pod)
	raw := k8s.Manager.IPtoResource(srcIP)

	// Try parsing Pod
	pod, isPod := raw.(*corev1.Pod)
	if isPod {
		return pod.Name, pod.Namespace, pod.Labels, "pod"
	}

	// Try parsing Service
	service, isService := raw.(*corev1.Service)
	if isService {
		return service.Name, service.Namespace, service.Labels, "service"
	}

	// We were not able to find the Kubernetes resource with given IP
	return name, namespace, labels, "Unknown"
}
