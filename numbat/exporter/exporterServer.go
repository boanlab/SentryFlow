package exporter

import (
	"log"
	"numbat/protobuf"
)

var exs *ExporterServer

// init Function
func init() {
	exs = NewExporterServer()
}

// ExporterServer Structure
type ExporterServer struct {
	protobuf.UnimplementedNumbatServer // @todo fix this
}

// NewExporterServer Function
func NewExporterServer() *ExporterServer {
	return new(ExporterServer)
}

// GetData Function
func (exs *ExporterServer) GetData(client *protobuf.ClientInfo, stream protobuf.Numbat_GetDataServer) error {
	log.Printf("[Exporter] Client %s(%s) connected", client.Hostname, client.Hostname)

outerLoop:
	for {
		select {
		case accessLog, ok := <-Exp.logChannel:
			if !ok {
				log.Printf("[Error] Unable to receive access log from channel")
			}

			// Send stream with replies
			// @todo: make max failure count for a single client
			curRetry := 0
			for curRetry < 3 { // @todo make this retry count configurable using configs
				err := stream.Send(accessLog)
				if err != nil {
					log.Printf("[Error] Unable to send access log to %s(%s) (retry=%d/%d): %v",
						client.Hostname, client.IpAddress, curRetry, 3, err)
					curRetry++
				} else {
					continue outerLoop
				}
			}

		case <-Exp.stopChan:
			return nil
		}
	}
}
