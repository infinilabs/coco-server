/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"time"
)

const (
	ModelProviderCachePrimary = "model_provider"
)

// GetModelProvider retrieves the model provider object from the cache or database.
func GetModelProvider(providerID string) (*core.ModelProvider, error) {
	item := GeneralObjectCache.Get(ModelProviderCachePrimary, providerID)
	var provider *core.ModelProvider
	if item != nil && !item.Expired() {
		var ok bool
		if provider, ok = item.Value().(*core.ModelProvider); ok {
			return provider, nil
		}
	}
	provider = &core.ModelProvider{}
	provider.ID = providerID
	_, err := orm.Get(provider)
	if err != nil {
		return nil, err
	}
	// Cache the provider object
	GeneralObjectCache.Set(ModelProviderCachePrimary, providerID, provider, time.Duration(30)*time.Minute)
	return provider, nil
}
