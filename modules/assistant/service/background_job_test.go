/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package service

import (
	"context"
	"fmt"
	"testing"

	"infini.sh/coco/modules/common"
)

func TestBuildReplyEndPayloadIncludesTimeoutType(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantType   string
		wantReason string
	}{
		{
			name:       "timeout without source defaults to assistant generation",
			err:        nil,
			wantType:   common.ReplyEndTimeoutTypeAssistantGeneration,
			wantReason: common.ReplyEndReasonTimeout,
		},
		{
			name:       "assistant generation timeout",
			err:        context.DeadlineExceeded,
			wantType:   common.ReplyEndTimeoutTypeAssistantGeneration,
			wantReason: common.ReplyEndReasonTimeout,
		},
		{
			name:       "attachment processing timeout",
			err:        fmt.Errorf("%w: %w", errAttachmentProcessingTimeout, context.DeadlineExceeded),
			wantType:   common.ReplyEndTimeoutTypeAttachmentProcessing,
			wantReason: common.ReplyEndReasonTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := buildReplyEndPayload(common.ReplyEndReasonTimeout, tt.err)

			if got := payload["reason"]; got != tt.wantReason {
				t.Fatalf("unexpected reason: got %v, want %s", got, tt.wantReason)
			}
			if got := payload["type"]; got != tt.wantType {
				t.Fatalf("unexpected timeout type: got %v, want %s", got, tt.wantType)
			}
		})
	}
}

func TestBuildReplyEndPayloadNonTimeoutReasons(t *testing.T) {
	tests := []struct {
		name      string
		reason    string
		err       error
		wantError string
	}{
		{
			name:   "completed",
			reason: common.ReplyEndReasonCompleted,
		},
		{
			name:   "user cancelled",
			reason: common.ReplyEndReasonUserCancelled,
		},
		{
			name:      "error",
			reason:    common.ReplyEndReasonError,
			err:       fmt.Errorf("llm request failed"),
			wantError: "llm request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := buildReplyEndPayload(tt.reason, tt.err)

			if got := payload["reason"]; got != tt.reason {
				t.Fatalf("unexpected reason: got %v, want %s", got, tt.reason)
			}
			if _, ok := payload["type"]; ok {
				t.Fatalf("unexpected timeout type in %s payload: %v", tt.reason, payload["type"])
			}
			if tt.wantError == "" {
				if _, ok := payload["error"]; ok {
					t.Fatalf("unexpected error field in %s payload: %v", tt.reason, payload["error"])
				}
				return
			}
			if got := payload["error"]; got != tt.wantError {
				t.Fatalf("unexpected error field: got %v, want %s", got, tt.wantError)
			}
		})
	}
}

func TestDetermineExitReasonTreatsAttachmentTimeoutAsTimeout(t *testing.T) {
	err := fmt.Errorf("%w: %w", errAttachmentProcessingTimeout, context.DeadlineExceeded)

	if got := determineExitReason(context.Background(), err, ""); got != common.ReplyEndReasonTimeout {
		t.Fatalf("unexpected exit reason: got %s, want %s", got, common.ReplyEndReasonTimeout)
	}
}
