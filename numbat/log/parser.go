package log

import (
	"numbat/common"
	"numbat/core"
	protobuf "numbat/protobuf"
	"strconv"
	"strings"
)

// GenerateLogs Function
func GenerateLogs(logText string) []*protobuf.Log {
	// @todo this needs more optimization, this code is kind of messy
	// Create an array of AccessLogs for returning gRPC comm
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

		// Lookup using K8s API
		src := core.LookupNetworkedResource(srcIP)
		dst := core.LookupNetworkedResource(dstIP)

		// Create AccessLog in our gRPC format
		cur := protobuf.Log{
			TimeStamp:    timeStamp,
			Id:           0, //  do 0 for now, we are going to write it later
			SrcNamespace: src.Namespace,
			SrcName:      src.Name,
			SrcLabel:     src.Labels,
			SrcIP:        srcIP,
			SrcPort:      srcPort,
			SrcType:      common.K8sResourceTypeToString(src.Type),
			DstNamespace: dst.Namespace,
			DstName:      dst.Name,
			DstLabel:     dst.Labels,
			DstIP:        dstIP,
			DstPort:      dstPort,
			DstType:      common.K8sResourceTypeToString(dst.Type),
			Protocol:     protocolName,
			Method:       method,
			Path:         path,
			ResponseCode: resCode,
		}

		ret = append(ret, &cur)
	}

	return ret
}
