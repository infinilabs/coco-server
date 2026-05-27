package milvus

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
)

type connectionTester struct{}

func (t *connectionTester) TestConnection(ctx context.Context, config map[string]interface{}) error {
	address, _ := config["address"].(string)
	if address == "" {
		return fmt.Errorf("address is required")
	}

	username, _ := config["username"].(string)
	password, _ := config["password"].(string)
	dbName, _ := config["db_name"].(string)

	cfg := client.Config{
		Address:  address,
		Username: username,
		Password: password,
		DBName:   dbName,
	}

	c, err := client.NewClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to Milvus at %s: %w", address, err)
	}
	defer c.Close()

	collection, _ := config["collection"].(string)
	if collection != "" {
		exists, err := c.HasCollection(ctx, collection)
		if err != nil {
			return fmt.Errorf("failed to check collection: %w", err)
		}
		if !exists {
			return fmt.Errorf("collection %q does not exist", collection)
		}
	}

	return nil
}
