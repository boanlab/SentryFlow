package client

import (
	pb "SentryFlow/protobuf"
	"context"
	"io"
	"log"
	"neo4j-client/neo4jdb"
)

// Feeder Structure
type Feeder struct {
	Running bool

	client    pb.SentryFlowClient
	logStream pb.SentryFlow_GetAPILogClient

	DbHandler neo4jdb.Neo4jHandler

	Done chan struct{}
}

// NewClient Function
func NewClient(client pb.SentryFlowClient, clientInfo *pb.ClientInfo, nodeLevel string, edgeLevel string, neo4jHost string, neo4jId string, neo4jPassword string) *Feeder {
	fd := &Feeder{}

	fd.Running = true
	fd.client = client
	fd.Done = make(chan struct{})

	// Contact the server and print out its response
	logStream, err := client.GetAPILog(context.Background(), clientInfo)
	if err != nil {
		log.Fatalf("[Client] Could not get API log: %v", err)
	}

	fd.logStream = logStream

	// Initialize DB
	dbHandler, err := neo4jdb.NewNeo4jHandler(nodeLevel, edgeLevel, neo4jHost, neo4jId, neo4jPassword)
	if err != nil {
		log.Fatalf("[MongoDB] Unable to intialize DB: %v", err)
		return nil
	}
	fd.DbHandler = *dbHandler

	return fd
}

// APILogRoutine Function
func (fd *Feeder) APILogRoutine() {
	for fd.Running {
		select {
		default:
			data, err := fd.logStream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("failed to receive log: %v", err)
			}
			log.Printf("[Client] Inserting log")
			err = fd.DbHandler.CreateOrUpdateRelationship(data)
			if err != nil {
				log.Printf("[Client] Failed to insert log: %v", err)
			}
		case <-fd.Done:
			return
		}
	}
}
