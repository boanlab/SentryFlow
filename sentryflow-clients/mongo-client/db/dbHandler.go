// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	protobuf "sentryflow/protobuf"
	"time"
)

type Handler struct {
	client            *mongo.Client
	database          *mongo.Database
	alCollection      *mongo.Collection
	metricsCollection *mongo.Collection
	cancel            context.CancelFunc
	dbURL             string
}

var Manager *Handler

// New creates a new mongoDB handler
func New() (*Handler, error) {
	dbHost := os.Getenv("MONGODB_HOST")
	h := Handler{}
	var err error

	// Environment variable was not set
	if dbHost == "" {
		return nil, errors.New("$MONGODB_HOST not set")
	}

	// Create a MongoDB client
	h.client, err = mongo.NewClient(options.Client().ApplyURI(dbHost))
	if err != nil {
		msg := fmt.Sprintf("unable to initialize monogoDB client for %s: %v", dbHost, err)
		return nil, errors.New(msg)
	}

	// Set timeout (10 sec)
	var ctx context.Context
	ctx, h.cancel = context.WithTimeout(context.Background(), 10*time.Second)

	// Try connecting the server
	err = h.client.Connect(ctx)
	if err != nil {
		msg := fmt.Sprintf("unable to connect mongoDB server %s: %v", dbHost, err)
		return nil, errors.New(msg)
	}

	// Create 'sentryflow' database and 'api-logs' collection
	h.database = h.client.Database("sentryflow")
	h.alCollection = h.database.Collection("api-logs")
	h.metricsCollection = h.database.Collection("metrics")

	Manager = &h
	return &h, nil
}

func (h *Handler) Disconnect() {
	err := h.client.Disconnect(context.Background())
	if err != nil {
		log.Printf("unable to properly disconnect: %v", err)
	}

	return
}

func (h *Handler) InsertAl(data *protobuf.APILog) error {
	_, err := h.alCollection.InsertOne(context.Background(), data)
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) InsertMetrics(data *protobuf.EnvoyMetric) error {
	_, err := h.metricsCollection.InsertOne(context.Background(), data)
	if err != nil {
		return err
	}

	return nil
}
