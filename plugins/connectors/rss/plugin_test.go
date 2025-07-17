/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// file: plugins/connectors/rss/plugin_test.go
package rss

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

func bytesToBase64String(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

type cache map[string][]byte

type mockKVStore struct {
	store map[string]cache
}

func (s *mockKVStore) Open() error {
	s.store = make(map[string]cache)
	return nil
}

func (s *mockKVStore) Close() error {
	return nil
}

func (s *mockKVStore) GetValue(bucket string, key []byte) ([]byte, error) {
	target := s.store[bucket]
	return target[bytesToBase64String(key)], nil
}

func (s *mockKVStore) GetCompressedValue(bucket string, key []byte) ([]byte, error) {
	return s.GetValue(bucket, key)
}

func (s *mockKVStore) AddValueCompress(bucket string, key []byte, value []byte) error {
	return s.AddValue(bucket, key, value)
}

func (s *mockKVStore) AddValue(bucket string, key []byte, value []byte) error {
	target := s.store[bucket]
	if target == nil {
		target = cache{}
		s.store[bucket] = target
	}
	target[bytesToBase64String(key)] = value
	return nil
}

func (s *mockKVStore) ExistsKey(bucket string, key []byte) (bool, error) {
	target := s.store[bucket]
	if target == nil {
		return false, nil
	}
	return target[bytesToBase64String(key)] != nil, nil
}

func (s *mockKVStore) DeleteKey(bucket string, key []byte) error {
	delete(s.store[bucket], bytesToBase64String(key))
	return nil
}

type mockQueue map[string][][]byte

func (q mockQueue) Name() string {
	return "indexing_documents"
}

func (q mockQueue) Init(s string) error {
	q[s] = [][]byte{}
	return nil
}

func (q mockQueue) Close(s string) error {
	q[s] = nil
	return nil
}

func (q mockQueue) GetStorageSize(k string) uint64 {
	return uint64(len(q[k]))
}

func (q mockQueue) Destroy(s string) error {
	clear(q[s])
	return nil
}

func (q mockQueue) GetQueues() []string {
	var ret []string
	for name := range q {
		ret = append(ret, name)
	}
	return ret
}

func (q mockQueue) Push(s string, bytes []byte) error {
	q[s] = append(q[s], bytes)
	return nil
}

// A simple RSS source for test
const sampleRSSFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="https://purl.org/rss/1.0/modules/content/">
<channel>
  <title>Test Feed</title>
  <item>
    <title>Test Item 1</title>
    <link>https://example.com/item1</link>
    <description>Summary for item 1.</description>
    <content:encoded><![CDATA[<p>Full content for item 1.</p>]]></content:encoded>
    <author>test@example.com (Test Author)</author>
    <guid>guid1</guid>
    <pubDate>Tue, 10 Jun 2024 04:00:00 MST</pubDate>
  </item>
  <item>
    <title>Test Item 2</title>
    <link>https://example.com/item2</link>
    <description>Summary for item 2.</description>
    <guid>guid2</guid>
    <pubDate>Wed, 11 Jun 2024 05:00:00 MST</pubDate>
  </item>
</channel>
</rss>`

func TestFetchRssFeed_Success(t *testing.T) {
	// mock Queue
	theQueue := mockQueue{}
	queue.RegisterDefaultHandler(theQueue)

	// mock KVStore
	kv.Register("indexing_documents", &mockKVStore{store: make(map[string]cache)})

	// A mock http server that provides RSS feeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleRSSFeed))
	}))
	defer server.Close()

	//  construct plugin & mock datasource
	testQueueName := "indexing_documents"
	plugin := &Plugin{
		Queue: &queue.QueueConfig{Name: testQueueName},
	}
	module.RegisterUserPlugin(plugin)
	plugin.Queue = queue.SmartGetOrInitConfig(plugin.Queue)

	connector := &common.Connector{}
	connector.ID = "rss"

	dataSource := &common.DataSource{}
	dataSource.ID = "test-datasource-id"
	dataSource.Name = "Test RSS Source"
	dataSource.Connector = common.ConnectorConfig{
		ConnectorID: "rss",
		Config: map[string]interface{}{
			"urls": []string{server.URL}, // mock server's URL
		},
	}

	// Defensive testing
	didPanic := false
	var panicValue interface{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				panicValue = r
			}
		}()
		// protected code
		plugin.fetchRssFeed(connector, dataSource)
	}()

	// --- Assert ---
	assert.False(t, didPanic, fmt.Sprintf("fetchRssFeed panicked with: %v", panicValue))

	queueID := plugin.Queue.ID
	// check queue size
	queueSize := len(theQueue[queueID])
	assert.Equal(t, 2, queueSize, "Expected 2 documents to be pushed to the queue")

	// mock queue's Pop operation
	data := theQueue[queueID][0]
	theQueue[queueID] = theQueue[queueID][1:]

	var doc1 common.Document
	err := json.Unmarshal(data, &doc1)
	assert.NoError(t, err)

	expectedID := util.MD5digest(fmt.Sprintf("%s-%s-%s", connector.ID, dataSource.ID, "guid1"))
	assert.Equal(t, "Test Item 1", doc1.Title)
	assert.Equal(t, "Summary for item 1.", doc1.Summary)
	assert.Equal(t, "<p>Full content for item 1.</p>", doc1.Content)
	assert.Equal(t, "https://example.com/item1", doc1.URL)
	assert.Equal(t, expectedID, doc1.ID)
	assert.Equal(t, "Test Author", doc1.Owner.UserName)
	assert.Equal(t, "rss", doc1.Type) // "feed" is more better?
	assert.Equal(t, "rss", doc1.Icon)
	createdTime, _ := time.Parse(time.RFC1123, "Tue, 10 Jun 2024 04:00:00 MST")
	assert.Equal(t, createdTime.UnixMilli(), doc1.Created.UnixMilli())
}
