package gcloud

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
)

type DataStoreHandler struct {
	dsClient *datastore.Client
}

func NewDataStoreHandler(ctx context.Context, projectId string, dbId string) (*DataStoreHandler, error) {
	if strings.TrimSpace(projectId) == "" {
		return nil, fmt.Errorf("Invalid projectId")
	}
	if strings.TrimSpace(dbId) == "" {
		return nil, fmt.Errorf("Invalid database id")
	}

	client, err := datastore.NewClientWithDatabase(ctx, projectId, dbId)
	if err != nil {
		return nil, err
	}

	new_handler := DataStoreHandler{
		dsClient: client,
	}

	return &new_handler, nil
}

func (dsh *DataStoreHandler) CreateNewRecord(key *datastore.Key, record any) (*datastore.Key, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*6)
	defer cancel()

	id, err := dsh.dsClient.Put(ctx, key, record)
	if err != nil {
		log.Printf("Error on saving record: %s", err.Error())
		return nil, err
	}

	return id, nil
}
