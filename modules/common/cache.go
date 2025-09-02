/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	ccache "infini.sh/framework/lib/cache"
)

var GeneralObjectCache = ccache.Layered(ccache.Configure().MaxSize(10000).ItemsToPrune(100))
