/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package utils

import (
	"fmt"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/datasource"
)

// GetConnectorID returns the connector ID for the given document.
func GetConnectorID(doc *core.Document) (string, error) {
	if doc.Source.ID == "" {
		return "", fmt.Errorf("document has no datasource ID")
	}

	datasources, err := datasource.GetDatasourceByID([]string{doc.Source.ID})
	if err != nil {
		return "", fmt.Errorf("failed to get datasource: %w", err)
	}
	if len(datasources) != 1 {
		return "", fmt.Errorf("expected exactly 1 datasource, got %d", len(datasources))
	}

	ds := datasources[0]
	if ds.Connector.ConnectorID == "" {
		return "", fmt.Errorf("datasource has no connector ID")
	}

	return ds.Connector.ConnectorID, nil
}
