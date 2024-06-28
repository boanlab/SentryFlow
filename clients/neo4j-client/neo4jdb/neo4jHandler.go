// SPDX-License-Identifier: Apache-2.0

package neo4jdb

import (
	"log"

	pb "SentryFlow/protobuf"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Neo4jHandler struct {
	Driver  neo4j.Driver
	Session neo4j.Session

	NodeLevel string
	EdgeLevel string
}

// NewHandler creates a new Neo4j handler
func NewNeo4jHandler(nodeLevel string, edgeLevel string, dbURI string, dbUsername string, dbPassword string) (*Neo4jHandler, error) {
	var dbHandler Neo4jHandler

	// Create a new driver for Neo4j
	driver, err := neo4j.NewDriver(dbURI, neo4j.BasicAuth(dbUsername, dbPassword, ""))
	if err != nil {
		log.Fatalf("Error connecting Neo4j %s: %v", dbURI, err)
		return &dbHandler, err
	}

	// Create session for Neo4j
	session := driver.NewSession(neo4j.SessionConfig{})

	dbHandler.Driver = driver
	dbHandler.Session = session

	dbHandler.NodeLevel = nodeLevel
	dbHandler.EdgeLevel = edgeLevel

	return &dbHandler, nil
}

func (h *Neo4jHandler) Close() {
	h.Session.Close()
	h.Driver.Close()
}

func (h *Neo4jHandler) CreateOrUpdateRelationship(APILog *pb.APILog) error {
	var query string
	var srcName string
	var dstName string

	if h.NodeLevel == "simple" {
		srcName = APILog.SrcLabel["app"]
		dstName = APILog.DstLabel["app"]
	} else {
		srcName = APILog.SrcName
		dstName = APILog.DstName
	}

	if h.EdgeLevel == "simple" {
		query = `
			MERGE (src:Pod {name: $srcNamem, namespace: $srcNamespace})
			ON CREATE SET src.name = $srcName
			MERGE (dst:Pod {namespace: $dstNamespace, label: $dstLabel})
			ON CREATE SET dst.name = $dstName
			MERGE (src)-[r:CALLS {method: $method}]->(dst)
			ON CREATE SET r.weight = 1
			ON MATCH SET r.weight = r.weight + 1
		`
	} else {
		query = `
			MERGE (src:Pod {name: $srcNamem, namespace: $srcNamespace})
			ON CREATE SET src.name = $srcName
			MERGE (dst:Pod {namespace: $dstNamespace, label: $dstLabel})
			ON CREATE SET dst.name = $dstName
			MERGE (src)-[r:CALLS {method: $method, path: $path}]->(dst)
			ON CREATE SET r.weight = 1
			ON MATCH SET r.weight = r.weight + 1
		`
	}

	log.Printf("[HI] %v, %v", srcName, dstName)

	params := map[string]interface{}{
		"srcNamespace": APILog.SrcNamespace,
		"srcName":      srcName,
		"dstNamespace": APILog.DstNamespace,
		"dstName":      dstName,
		"method":       APILog.Method,
		"path":         APILog.Path,
	}

	_, err := h.Session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		return tx.Run(query, params)
	})

	return err
}
