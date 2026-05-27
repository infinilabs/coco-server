/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"context"
	"database/sql"
	"fmt"

	"infini.sh/coco/plugins/connectors"
)

// TestSQLConnection opens a SQL connection and pings the database to verify connectivity.
func TestSQLConnection(ctx context.Context, driverName, connectionURI string) error {
	if connectionURI == "" {
		return fmt.Errorf("connection_uri is required")
	}

	db, err := sql.Open(driverName, connectionURI)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, connectors.DefaultConnectionTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}
