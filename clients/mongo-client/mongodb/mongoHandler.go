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

// DBHandler Structure
type DBHandler struct {
	client *mongo.Client
	cancel context.CancelFunc

	database      *mongo.Database
	apiLogCol     *mongo.Collection
	apiMetricsCol *mongo.Collection
	evyMetricsCol *mongo.Collection
}

// dbHandler for Global Reference
var dbHandler DBHandler

// NewMongoDBHandler Function
func NewMongoDBHandler(mongoDBAddr string) (*DBHandler, error) {
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
	dbHandler.apiMetricsCol = dbHandler.database.Collection("APIMetrics")
	dbHandler.evyMetricsCol = dbHandler.database.Collection("EnvoyMetrics")

	return &dbHandler, nil
}

// Disconnect Function
func (handler *DBHandler) Disconnect() {
	err := handler.client.Disconnect(context.Background())
	if err != nil {
		log.Printf("[MongoDB] Unable to properly disconnect: %v", err)
	}
}

// InsertAPILog Function
func (handler *DBHandler) InsertAPILog(data *protobuf.APILog) error {
	_, err := handler.apiLogCol.InsertOne(context.Background(), data)
	if err != nil {
		return err
	}

	return nil
}

// InsertAPIMetrics Function
func (handler *DBHandler) InsertAPIMetrics(data *protobuf.APIMetrics) error {
	_, err := handler.apiMetricsCol.InsertOne(context.Background(), data)
	if err != nil {
		return err
	}

	return nil
}

// InsertEnvoyMetrics Function
func (handler *DBHandler) InsertEnvoyMetrics(data *protobuf.EnvoyMetrics) error {
	_, err := handler.evyMetricsCol.InsertOne(context.Background(), data)
	if err != nil {
		return err
	}

	return nil
}
