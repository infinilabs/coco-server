/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoffSchedule(t *testing.T) {
	assert.Equal(t, 2*time.Second, BackoffDelay(1))
	assert.Equal(t, 4*time.Second, BackoffDelay(2))
	assert.Equal(t, 8*time.Second, BackoffDelay(3))
	assert.Equal(t, 8*time.Second, BackoffDelay(9), "capped at 8s")
}

func TestIsRetryableStatus(t *testing.T) {
	assert.True(t, IsRetryableStatus(429))
	assert.True(t, IsRetryableStatus(500))
	assert.True(t, IsRetryableStatus(503))
	assert.False(t, IsRetryableStatus(404))
	assert.False(t, IsRetryableStatus(401))
}

func TestRetryWithBackoffSucceeds(t *testing.T) {
	calls := 0
	err := RetryWithBackoff(context.Background(), func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestRetryWithBackoffExhausts(t *testing.T) {
	calls := 0
	err := RetryWithBackoff(context.Background(), func() error {
		calls++
		return errors.New("always failing")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "giving up after 3 attempts")
	assert.Equal(t, 3, calls)
}

func TestRetryWithBackoffHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := RetryWithBackoff(ctx, func() error {
		calls++
		return nil
	})
	assert.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, calls, "a cancelled context never calls fn")
}
