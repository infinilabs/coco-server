/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package oracle

import (
	"context"
	"fmt"

	_ "github.com/sijms/go-ora/v2"
	cmn "infini.sh/coco/plugins/connectors/common"
)

type connectionTester struct{}

func (t *connectionTester) TestConnection(ctx context.Context, config map[string]interface{}) error {
	uri, _ := config["connection_uri"].(string)
	if uri == "" {
		return fmt.Errorf("connection_uri is required")
	}
	return cmn.TestSQLConnection(ctx, "oracle", uri)
}
