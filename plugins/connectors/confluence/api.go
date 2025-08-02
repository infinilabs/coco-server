/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package confluence

import "context"

type ContentAPI interface {

	// SearchContent utilizes the Confluence API: /rest/api/content/search
	// Link: https://developer.atlassian.com/cloud/confluence/rest/v1/api-group-content/#api-wiki-rest-api-content-search-get
	SearchContent(ctx context.Context, query SearchContentRequest, options ...QueryOption) (*SearchContentResponse, error)

	// SearchNextContent utilizes the `next` capability provided by the Confluence API,/rest/api/content/search to search for results.
	// Link: https://developer.atlassian.com/cloud/confluence/rest/v1/api-group-content/#api-wiki-rest-api-content-search-get
	SearchNextContent(ctx context.Context, next string) (*SearchContentResponse, error)
}
