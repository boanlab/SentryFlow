// SPDX-License-Identifier: Apache-2.0

package client

import (
	pb "SentryFlow/protobuf"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Feeder Structure
type Feeder struct {
	Running bool

	client            pb.SentryFlowClient
	logStream         pb.SentryFlow_GetAPILogClient
	envoyMetricStream pb.SentryFlow_GetEnvoyMetricsClient
	apiMetricStream   pb.SentryFlow_GetAPIMetricsClient

	Done chan struct{}
}

// StrToFile Function
func StrToFile(str, targetFile string) {
	_, err := os.Stat(targetFile)
	if err != nil {
		newFile, err := os.Create(filepath.Clean(targetFile))
		if err != nil {
			fmt.Printf("[Client] Failed to create a file (%s, %s)\n", targetFile, err.Error())
			return
		}
		err = newFile.Close()
		if err != nil {
			fmt.Printf("[Client] Failed to close the file (%s, %s)\n", targetFile, err.Error())
		}
	}

	file, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Printf("[Client] Failed to open a file (%s, %s)\n", targetFile, err.Error())
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("[Client] Failed to close the file (%s, %s)\n", targetFile, err.Error())
		}
	}()

	_, err = file.WriteString(str)
	if err != nil {
		fmt.Printf("[Client] Failed to write a string into the file (%s, %s)\n", targetFile, err.Error())
	}
}

// NewClient Function
func NewClient(client pb.SentryFlowClient, clientInfo *pb.ClientInfo, logCfg string, metricCfg string, metricFilter string) *Feeder {
	fd := &Feeder{}

	fd.Running = true

	fd.client = client

	fd.Done = make(chan struct{})

	if logCfg != "none" {
		// Contact the server and print out its response
		logStream, err := client.GetAPILog(context.Background(), clientInfo)
		if err != nil {
			log.Fatalf("[Client] Could not get log: %v", err)
		}

		fd.logStream = logStream
	}

	if metricCfg != "none" && (metricFilter == "all" || metricFilter == "api") {
		amStream, err := client.GetAPIMetrics(context.Background(), clientInfo)
		if err != nil {
			log.Fatalf("[Client] Could not get log: %v", err)
		}

		fd.apiMetricStream = amStream
	}

	if metricCfg != "none" && (metricFilter == "all" || metricFilter == "envoy") {
		emStream, err := client.GetEnvoyMetrics(context.Background(), clientInfo)
		if err != nil {
			log.Fatalf("[Client] Could not get log: %v", err)
		}

		fd.envoyMetricStream = emStream
	}

	return fd
}

// LogRoutine Function
func (fd *Feeder) LogRoutine(logCfg string) {
	for fd.Running {
		select {
		default:
			data, err := fd.logStream.Recv()
			if err != nil {
				log.Fatalf("[Client] Failed to receive a log: %v", err)
				break
			}
			str := ""
			str = str + "== Access Log ==\n"
			str = str + fmt.Sprintf("%v\n", data)

			if logCfg == "stdout" {
				fmt.Printf("%s", str)
			} else {
				StrToFile(str, logCfg)
			}
		case <-fd.Done:
			return
		}
	}
}

// APIMetricRoutine Function
func (fd *Feeder) APIMetricRoutine(metricCfg string) {
	for fd.Running {
		select {
		default:
			data, err := fd.apiMetricStream.Recv()
			if err != nil {
				log.Fatalf("[Client] Failed to receive metrics: %v", err)
				break
			}

			str := ""
			str = str + "== API Metrics ==\n"
			str = str + fmt.Sprintf("%v\n", data)

			if metricCfg == "stdout" {
				fmt.Printf("%s", str)
			} else {
				StrToFile(str, metricCfg)
			}
		case <-fd.Done:
			return
		}
	}
}

// EnvoyMetricRoutine Function
func (fd *Feeder) EnvoyMetricRoutine(metricCfg string) {
	metricKeys := []string{"GAUGE", "COUNTER", "HISTOGRAM", "SUMMARY"}
	for fd.Running {
		select {
		default:
			data, err := fd.envoyMetricStream.Recv()
			if err != nil {
				log.Fatalf("[Client] Failed to receive metrics: %v", err)
				break
			}

			str := ""
			str = fmt.Sprintf("== Envoy Metrics / %s ==\n", data.TimeStamp)
			str = str + fmt.Sprintf("Namespace: %s\n", data.Namespace)
			str = str + fmt.Sprintf("Name: %s\n", data.Name)
			str = str + fmt.Sprintf("IPAddress: %s\n", data.IPAddress)
			str = str + fmt.Sprintf("Labels: %s\n", data.Labels)

			for _, key := range metricKeys {
				str = str + fmt.Sprintf("%s: {%v}\n", key, data.Metrics[key])
			}

			if metricCfg == "stdout" {
				fmt.Printf("%s", str)
			} else {
				StrToFile(str, metricCfg)
			}
		case <-fd.Done:
			return
		}
	}
}
