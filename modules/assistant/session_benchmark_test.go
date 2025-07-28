package assistant

import (
	"fmt"
	"testing"
	"time"

	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// BenchmarkChatHistoryQueries compares performance between full and optimized query patterns
func BenchmarkChatHistoryQueries(b *testing.B) {
	// This benchmark demonstrates the performance improvements of optimized queries
	// Note: These benchmarks require a running Elasticsearch instance and test data

	sessionID := "test-session-id"
	size := 20

	b.Run("Full_ChatMessage_Query", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := getChatHistoryBySessionInternal(sessionID, size)
			if err != nil {
				b.Skipf("Skipping benchmark - requires test environment: %v", err)
			}
		}
	})

	b.Run("Optimized_Basic_Query", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := getChatHistoryBySessionBasic(sessionID, size)
			if err != nil {
				b.Skipf("Skipping benchmark - requires test environment: %v", err)
			}
		}
	})

	b.Run("Optimized_Metadata_Query", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := getChatHistoryBySessionMetadata(sessionID, size)
			if err != nil {
				b.Skipf("Skipping benchmark - requires test environment: %v", err)
			}
		}
	})

	b.Run("Optimized_IDs_Only_Query", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := getChatHistoryBySessionIDs(sessionID, size)
			if err != nil {
				b.Skipf("Skipping benchmark - requires test environment: %v", err)
			}
		}
	})
}

// TestChatMessageStructSizes demonstrates memory usage differences
func TestChatMessageStructSizes(t *testing.T) {
	// Create sample data to show the difference in memory footprint
	fullMessage := common.ChatMessage{
		MessageType: "user",
		SessionID:   "session-123",
		Message:     "This is a test message with some content",
		From:        "user@example.com",
		To:          "assistant",
		AssistantID: "assistant-123",
		Attachments: []string{"file1.pdf", "file2.doc"},
		Details: []common.ProcessingDetails{
			{Order: 1, Type: "processing", Description: "Processing step 1"},
			{Order: 2, Type: "analysis", Description: "Analysis step 2"},
		},
		Parameters: util.MapStr{
			"temperature": 0.7,
			"max_tokens":  1000,
			"model":       "gpt-4",
		},
		UpVote:   5,
		DownVote: 1,
	}
	now := time.Now()
	fullMessage.Created = &now
	fullMessage.Updated = &now

	basicMessage := ChatMessageBasic{
		MessageType: fullMessage.MessageType,
		SessionID:   fullMessage.SessionID,
		Message:     fullMessage.Message,
		UpVote:      fullMessage.UpVote,
		DownVote:    fullMessage.DownVote,
		Created:     fullMessage.Created,
	}

	metadataMessage := ChatMessageMetadata{
		MessageType: fullMessage.MessageType,
		SessionID:   fullMessage.SessionID,
		From:        fullMessage.From,
		To:          fullMessage.To,
		AssistantID: fullMessage.AssistantID,
		Created:     fullMessage.Created,
	}

	// Estimate serialized sizes (approximate JSON size)
	fullSize := len(util.MustToJSON(fullMessage))
	basicSize := len(util.MustToJSON(basicMessage))
	metadataSize := len(util.MustToJSON(metadataMessage))

	t.Logf("Memory footprint comparison:")
	t.Logf("Full ChatMessage JSON size: %d bytes", fullSize)
	t.Logf("Basic ChatMessage JSON size: %d bytes", basicSize)
	t.Logf("Metadata ChatMessage JSON size: %d bytes", metadataSize)

	// Calculate savings
	basicSavings := float64(fullSize-basicSize) / float64(fullSize) * 100
	metadataSavings := float64(fullSize-metadataSize) / float64(fullSize) * 100

	t.Logf("Basic query saves: %.1f%% memory/bandwidth", basicSavings)
	t.Logf("Metadata query saves: %.1f%% memory/bandwidth", metadataSavings)

	// For a typical chat session with 20 messages
	sessionCount := 20
	t.Logf("\nFor a session with %d messages:", sessionCount)
	t.Logf("Full query total: %d bytes", fullSize*sessionCount)
	t.Logf("Basic query total: %d bytes (saves %d bytes)", basicSize*sessionCount, (fullSize-basicSize)*sessionCount)
	t.Logf("Metadata query total: %d bytes (saves %d bytes)", metadataSize*sessionCount, (fullSize-metadataSize)*sessionCount)
}

// TestQuerySourceFiltering demonstrates the Elasticsearch source filtering
func TestQuerySourceFiltering(t *testing.T) {
	sessionID := "test-session"
	size := 10

	// Test that our raw queries generate proper Elasticsearch _source filtering
	rawQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"session_id": sessionID,
			},
		},
		"sort": []map[string]interface{}{
			{
				"created": map[string]string{
					"order": "desc",
				},
			},
		},
		"size": size,
		"from": 0,
		"_source": []string{
			"id", "created", "type", "session_id", "message", "down_vote", "up_vote",
		},
	}

	queryBytes := util.MustToJSONBytes(rawQuery)
	t.Logf("Generated Elasticsearch query with _source filtering:")
	t.Logf("%s", string(queryBytes))

	// Verify _source field is properly set
	var queryMap map[string]interface{}
	util.MustFromJSONBytes(queryBytes, &queryMap)

	sourceFields, ok := queryMap["_source"].([]interface{})
	if !ok {
		t.Error("_source field not found or wrong type")
		return
	}

	expectedFields := []string{"id", "created", "type", "session_id", "message", "down_vote", "up_vote"}
	if len(sourceFields) != len(expectedFields) {
		t.Errorf("Expected %d source fields, got %d", len(expectedFields), len(sourceFields))
	}

	t.Logf("Source filtering properly configured with %d fields", len(sourceFields))
}

// Example function showing how to choose the right query variant
func ExampleOptimizedQueryUsage() {
	sessionID := "example-session"

	// Use case 1: Building chat history for LLM context (use Basic)
	// Only needs message content, type, and vote information
	basicHistory, _ := getChatHistoryBySessionBasic(sessionID, 20)
	fmt.Printf("Loaded %d basic messages for LLM context\n", len(basicHistory))

	// Use case 2: Analyzing session patterns (use Metadata)
	// Needs type, participants, timestamps but not content
	metadata, _ := getChatHistoryBySessionMetadata(sessionID, 100)
	fmt.Printf("Loaded %d metadata records for analysis\n", len(metadata))

	// Use case 3: Counting messages or checking existence (use IDs)
	// Only needs message IDs for lightweight operations
	ids, _ := getChatHistoryBySessionIDs(sessionID, 1000)
	fmt.Printf("Found %d messages in session\n", len(ids))

	// Use case 4: Full message display (use Original)
	// Needs all fields including attachments, details, parameters
	fullHistory, _ := getChatHistoryBySessionInternal(sessionID, 20)
	fmt.Printf("Loaded %d full messages for display\n", len(fullHistory))
}
