/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package mysql

import (
	"context"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	cmn "infini.sh/coco/plugins/connectors/common"
)

type connectionTester struct{}

func (t *connectionTester) TestConnection(ctx context.Context, config map[string]interface{}) error {
	uri, _ := config["connection_uri"].(string)
	if uri == "" {
		return fmt.Errorf("connection_uri is required")
	}
	return cmn.TestSQLConnection(ctx, "mysql", uri)
}
