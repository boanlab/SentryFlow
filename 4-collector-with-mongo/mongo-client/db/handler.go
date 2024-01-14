package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"mongo-client/common"
	"os"
	"time"
)

type Handler struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
	context    context.Context
	cancel     context.CancelFunc
	dbURL      string
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

	// Yes this is deprecated, but we are just doing a demo
	h.client, err = mongo.NewClient(options.Client().ApplyURI(dbHost))
	if err != nil {
		msg := fmt.Sprintf("unable to initialize monogoDB client for %s: %v", dbHost, err)
		return nil, errors.New(msg)
	}

	// Set timeout as 10 sec
	h.context, h.cancel = context.WithTimeout(context.Background(), 10*time.Second)

	// Try connecting server
	err = h.client.Connect(h.context)
	if err != nil {
		msg := fmt.Sprintf("unable to connect mongoDB server %s: %v", dbHost, err)
		return nil, errors.New(msg)
	}

	// Create database of istio-otel and collection of access-logs
	h.database = h.client.Database("istio-otel")
	h.collection = h.database.Collection("access-logs")

	Manager = &h
	return &h, nil
}

func (h *Handler) Disconnect() {
	err := h.client.Disconnect(h.context)
	if err != nil {
		log.Printf("unable to properly disconnect: %v", err)
	}

	return
}

func (h *Handler) InsertData(data common.MyLog) error {
	data.TimeStamp = time.Now()
	data.UUID = uuid.New()

	_, err := h.collection.InsertOne(h.context, data)
	if err != nil {
		return err
	}

	return nil
}
