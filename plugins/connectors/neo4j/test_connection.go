package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type connectionTester struct{}

func (t *connectionTester) TestConnection(ctx context.Context, config map[string]interface{}) error {
	uri, _ := config["connection_uri"].(string)
	if uri == "" {
		return fmt.Errorf("connection_uri is required")
	}

	authToken, _ := config["auth_token"].(string)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)

	auth := neo4j.NoAuth()
	if authToken != "" {
		auth = neo4j.BearerAuth(authToken)
	} else if username != "" || password != "" {
		auth = neo4j.BasicAuth(username, password, "")
	}

	driver, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		return fmt.Errorf("failed to create Neo4j driver: %w", err)
	}
	defer driver.Close(ctx)

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	return nil
}
