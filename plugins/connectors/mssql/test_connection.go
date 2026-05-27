/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/microsoft/go-mssqldb"
	"infini.sh/coco/plugins/connectors"
)

type connectionTester struct{}

func (t *connectionTester) TestConnection(ctx context.Context, config map[string]interface{}) error {
	uri, _ := config["connection_uri"].(string)
	if uri == "" {
		return fmt.Errorf("connection_uri is required")
	}

	expectedDB := databaseFromMSSQLURI(uri)

	db, err := sql.Open("mssql", uri)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, connectors.DefaultConnectionTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// go-mssqldb silently falls back to `master` when the configured database
	// does not exist or the user has no access, so Ping alone cannot detect a
	// wrong database name. Verify the actual connected database matches.
	if expectedDB != "" {
		var current sql.NullString
		if err := db.QueryRowContext(pingCtx, "SELECT DB_NAME()").Scan(&current); err != nil {
			return fmt.Errorf("failed to verify connected database: %w", err)
		}
		if !strings.EqualFold(current.String, expectedDB) {
			return fmt.Errorf("database %q is not accessible (connected to %q instead)", expectedDB, current.String)
		}
	}

	return nil
}

// databaseFromMSSQLURI extracts the database name from any of the connection
// string forms accepted by github.com/microsoft/go-mssqldb. Returns "" if no
// database is specified.
//
//	sqlserver://user:pass@host:port/dbname
//	sqlserver://user:pass@host:port?database=dbname
//	server=host;user id=sa;password=...;database=dbname
func databaseFromMSSQLURI(uri string) string {
	lower := strings.ToLower(uri)
	if strings.HasPrefix(lower, "sqlserver://") {
		if u, err := url.Parse(uri); err == nil {
			if path := strings.TrimPrefix(u.Path, "/"); path != "" {
				return path
			}
			if v := u.Query().Get("database"); v != "" {
				return v
			}
		}
		return ""
	}
	for _, kv := range strings.Split(uri, ";") {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(kv[:eq])
		v := strings.TrimSpace(kv[eq+1:])
		if strings.EqualFold(k, "database") || strings.EqualFold(k, "initial catalog") {
			return v
		}
	}
	return ""
}
