/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"context"
	"sync"
)

// ConnectionTester is an optional interface that connectors can implement
// to support testing connection configuration before saving a datasource.
type ConnectionTester interface {
	TestConnection(ctx context.Context, config map[string]interface{}) error
}

var (
	connectionTesters   = map[string]ConnectionTester{}
	connectionTestersMu sync.RWMutex
)

func RegisterConnectionTester(name string, tester ConnectionTester) {
	connectionTestersMu.Lock()
	defer connectionTestersMu.Unlock()
	connectionTesters[name] = tester
}

func GetConnectionTester(name string) (ConnectionTester, bool) {
	connectionTestersMu.RLock()
	defer connectionTestersMu.RUnlock()
	tester, ok := connectionTesters[name]
	return tester, ok
}
