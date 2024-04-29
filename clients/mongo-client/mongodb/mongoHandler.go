// SPDX-License-Identifier: Apache-2.0

package mongodb

import (
	protobuf "SentryFlow/protobuf"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBHandler Structure
type MongoDBHandler struct {
	client *mongo.Client
	cancel context.CancelFunc

	database   *mongo.Database
	apiLogCol  *mongo.Collection
	metricsCol *mongo.Collection
}

// dbHandler structure
var dbHandler MongoDBHandler

// New creates a new mongoDB handler
func NewMongoDBHandler(mongoDBAddr string) (*MongoDBHandler, error) {
	var err error

	// Create a MongoDB client
	dbHandler.client, err = mongo.NewClient(options.Client().ApplyURI(mongoDBAddr))
	if err != nil {
		msg := fmt.Sprintf("[MongoDB] Unable to initialize a monogoDB client (%s): %v", mongoDBAddr, err)
		return nil, errors.New(msg)
	}

	// Set timeout (10 sec)
	var ctx context.Context
	ctx, dbHandler.cancel = context.WithTimeout(context.Background(), 10*time.Second)

	// Connect to the MongoDB server
	err = dbHandler.client.Connect(ctx)
	if err != nil {
		msg := fmt.Sprintf("[MongoDB] Unable to connect the mongoDB server (%s): %v", mongoDBAddr, err)
		return nil, errors.New(msg)
	}

	// Create 'SentryFlow' database
	dbHandler.database = dbHandler.client.Database("SentryFlow")

	// Create APILogs and Metrics collections
	dbHandler.apiLogCol = dbHandler.database.Collection("APILogs")
	dbHandler.metricsCol = dbHandler.database.Collection("Metrics")

	return &dbHandler, nil
}

// Disconnect function
func (handler *MongoDBHandler) Disconnect() {
	err := handler.client.Disconnect(context.Background())
	if err != nil {
		log.Printf("[MongoDB] Unable to properly disconnect: %v", err)
	}
}

// InsertAl function
func (handler *MongoDBHandler) InsertAPILog(data *protobuf.APILog) error {
	_, err := handler.apiLogCol.InsertOne(context.Background(), data)
	if err != nil {
		return err
	}

	return nil
}

// InsertMetrics function
func (handler *MongoDBHandler) InsertAPIMetrics(data *protobuf.APIMetric) error {
	_, err := handler.metricsCol.InsertOne(context.Background(), data)
	if err != nil {
		return err
	}

	return nil
}

// InsertMetrics function
func (handler *MongoDBHandler) InsertEnvoyMetrics(data *protobuf.EnvoyMetric) error {
	_, err := handler.metricsCol.InsertOne(context.Background(), data)
	if err != nil {
		return err
	}

	return nil
}
