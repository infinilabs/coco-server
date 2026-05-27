package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type connectionTester struct{}

func (t *connectionTester) TestConnection(ctx context.Context, config map[string]interface{}) error {
	uri, _ := config["connection_uri"].(string)
	if uri == "" {
		return fmt.Errorf("connection_uri is required")
	}

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer client.Disconnect(ctx)

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database, _ := config["database"].(string)
	if database != "" {
		databases, err := client.ListDatabaseNames(ctx, bson.M{"name": database})
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}
		if len(databases) == 0 {
			return fmt.Errorf("database %q does not exist", database)
		}

		collection, _ := config["collection"].(string)
		if collection != "" {
			collections, err := client.Database(database).ListCollectionNames(ctx, bson.M{"name": collection})
			if err != nil {
				return fmt.Errorf("failed to list collections: %w", err)
			}
			if len(collections) == 0 {
				return fmt.Errorf("collection %q does not exist in database %q", collection, database)
			}
		}
	}

	return nil
}
