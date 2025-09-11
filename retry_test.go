package avrocurio

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryMechanism(t *testing.T) {
	tests := []struct {
		name            string
		statusCodes     []int
		expectedRetries int
		expectSuccess   bool
	}{
		{
			name:            "immediate success",
			statusCodes:     []int{200},
			expectedRetries: 0,
			expectSuccess:   true,
		},
		{
			name:            "success after one retry",
			statusCodes:     []int{500, 200},
			expectedRetries: 1,
			expectSuccess:   true,
		},
		{
			name:            "success after max retries",
			statusCodes:     []int{500, 500, 500, 200},
			expectedRetries: 3,
			expectSuccess:   true,
		},
		{
			name:            "all attempts fail",
			statusCodes:     []int{500, 500, 500, 500},
			expectedRetries: 3,
			expectSuccess:   false,
		},
		{
			name:            "client error not retried",
			statusCodes:     []int{400},
			expectedRetries: 0,
			expectSuccess:   true, // Success in terms of getting a response
		},
		{
			name:            "timeout error retried",
			statusCodes:     []int{408, 200},
			expectedRetries: 1,
			expectSuccess:   true,
		},
		{
			name:            "rate limit error retried",
			statusCodes:     []int{429, 200},
			expectedRetries: 1,
			expectSuccess:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int64

			// Create test server that returns different status codes
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count := atomic.AddInt64(&requestCount, 1) - 1
				if int(count) < len(tt.statusCodes) {
					w.WriteHeader(tt.statusCodes[count])
					if tt.statusCodes[count] == 200 {
						_, _ = w.Write([]byte(`{}`))
					}
				} else {
					// Fallback to last status code if we run out
					w.WriteHeader(tt.statusCodes[len(tt.statusCodes)-1])
				}
			}))
			defer server.Close()

			// Create client with fast retry settings for testing
			config := NewApicurioConfig().
				WithBaseURL(server.URL).
				WithMaxRetries(3).
				WithRetryBaseDelay(1 * time.Millisecond).
				WithRetryMaxDelay(10 * time.Millisecond).
				WithRetryJitter(false) // Disable jitter for predictable testing

			client, err := NewApicurioClient(config)
			require.NoError(t, err)
			defer client.Close()

			ctx := context.Background()
			resp, err := client.makeRequest(ctx, "GET", "/test", nil)

			actualRetries := int(atomic.LoadInt64(&requestCount)) - 1

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if resp != nil {
					resp.Body.Close() //nolint:errcheck
				}
			} else {
				assert.Error(t, err)
			}

			assert.Equal(t, tt.expectedRetries, actualRetries, "unexpected number of retries")
		})
	}
}

func TestRetryBackoffStrategy(t *testing.T) {
	// Test that the backoff strategy is properly configured
	config := NewApicurioConfig().
		WithRetryBaseDelay(100 * time.Millisecond).
		WithRetryMaxDelay(5 * time.Second).
		WithRetryJitter(false)

	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	// We can't directly test the delay calculation since it's handled by the library,
	// but we can test that the configuration is valid and retry behavior works
	assert.Equal(t, 100*time.Millisecond, config.RetryBaseDelay)
	assert.Equal(t, 5*time.Second, config.RetryMaxDelay)
	assert.False(t, config.RetryJitterEnabled)
}

func TestRetryWithJitterConfiguration(t *testing.T) {
	// Test configuration with jitter enabled
	config := NewApicurioConfig().
		WithRetryBaseDelay(100 * time.Millisecond).
		WithRetryMaxDelay(5 * time.Second).
		WithRetryJitter(true)

	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	// Verify jitter is enabled in configuration
	assert.True(t, config.RetryJitterEnabled)
}

func TestShouldRetryStatusCode(t *testing.T) {
	client := &ApicurioClient{}

	tests := []struct {
		statusCode  int
		shouldRetry bool
	}{
		// Success codes - don't retry
		{200, false},
		{201, false},
		{204, false},

		// Redirect codes - don't retry
		{301, false},
		{302, false},

		// Client errors - mostly don't retry
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{408, true}, // Request Timeout - retry
		{429, true}, // Too Many Requests - retry

		// Server errors - always retry
		{500, true},
		{501, true},
		{502, true},
		{503, true},
		{504, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			result := client.shouldRetryStatusCode(tt.statusCode)
			assert.Equal(t, tt.shouldRetry, result)
		})
	}
}

func TestRetryWithContextCancellation(t *testing.T) {
	// Create server that always returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // Simulate slow response
		w.WriteHeader(500)
	}))
	defer server.Close()

	config := NewApicurioConfig().
		WithBaseURL(server.URL).
		WithMaxRetries(5).
		WithRetryBaseDelay(100 * time.Millisecond)

	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = client.makeRequest(ctx, "GET", "/test", nil)
	elapsed := time.Since(start)

	// Should fail with context error
	assert.Error(t, err)
	// Check for context deadline exceeded error specifically
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Should not wait for all retries - should cancel quickly
	assert.Less(t, elapsed, 500*time.Millisecond, "should cancel before completing all retries")
}
