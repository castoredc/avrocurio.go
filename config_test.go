package avrocurio

import (
	"strings"
	"testing"
	"time"
)

const (
	testUsername = "user"
	testPassword = "pass"
)

func TestNewApicurioConfig(t *testing.T) {
	config := NewApicurioConfig()

	// Test default values
	if config.BaseURL != defaultBaseURL {
		t.Errorf("Default BaseURL = %v, want http://localhost:8080", config.BaseURL)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Default Timeout = %v, want 30s", config.Timeout)
	}

	if config.MaxRetries != 3 {
		t.Errorf("Default MaxRetries = %v, want 3", config.MaxRetries)
	}

	if config.SchemaCacheSize != 1000 {
		t.Errorf("Default SchemaCacheSize = %v, want 1000", config.SchemaCacheSize)
	}

	if config.FailedLookupCacheSize != 100 {
		t.Errorf("Default FailedLookupCacheSize = %v, want 100", config.FailedLookupCacheSize)
	}

	if config.FailedLookupCacheTTL != 5*time.Minute {
		t.Errorf("Default FailedLookupCacheTTL = %v, want 5m", config.FailedLookupCacheTTL)
	}

	if config.RetryBaseDelay != 100*time.Millisecond {
		t.Errorf("Default RetryBaseDelay = %v, want 100ms", config.RetryBaseDelay)
	}

	if config.RetryMaxDelay != 30*time.Second {
		t.Errorf("Default RetryMaxDelay = %v, want 30s", config.RetryMaxDelay)
	}

	if !config.RetryJitterEnabled {
		t.Errorf("Default RetryJitterEnabled = %v, want true", config.RetryJitterEnabled)
	}

	// Test that auth is not set by default
	if config.HasBasicAuth() {
		t.Errorf("Default config should not have basic auth")
	}
}

func TestApicurioConfig_WithMethods(t *testing.T) {
	config := NewApicurioConfig()

	// Test WithBaseURL
	config = config.WithBaseURL("https://registry.example.com")
	if config.BaseURL != "https://registry.example.com" {
		t.Errorf("WithBaseURL failed: got %v", config.BaseURL)
	}

	// Test WithTimeout
	config = config.WithTimeout(60 * time.Second)
	if config.Timeout != 60*time.Second {
		t.Errorf("WithTimeout failed: got %v", config.Timeout)
	}

	// Test WithMaxRetries
	config = config.WithMaxRetries(5)
	if config.MaxRetries != 5 {
		t.Errorf("WithMaxRetries failed: got %v", config.MaxRetries)
	}

	// Test WithBasicAuth
	config = config.WithBasicAuth(testUsername, testPassword)
	if config.Username != testUsername || config.Password != testPassword {
		t.Errorf("WithBasicAuth failed: got %v:%v", config.Username, config.Password)
	}

	if !config.HasBasicAuth() {
		t.Errorf("HasBasicAuth should return true after setting auth")
	}

	// Test WithSchemaCacheSize
	config = config.WithSchemaCacheSize(2000)
	if config.SchemaCacheSize != 2000 {
		t.Errorf("WithSchemaCacheSize failed: got %v", config.SchemaCacheSize)
	}

	// Test WithFailedLookupCacheSize
	config = config.WithFailedLookupCacheSize(200)
	if config.FailedLookupCacheSize != 200 {
		t.Errorf("WithFailedLookupCacheSize failed: got %v", config.FailedLookupCacheSize)
	}

	// Test WithFailedLookupCacheTTL
	config = config.WithFailedLookupCacheTTL(10 * time.Minute)
	if config.FailedLookupCacheTTL != 10*time.Minute {
		t.Errorf("WithFailedLookupCacheTTL failed: got %v", config.FailedLookupCacheTTL)
	}

	// Test WithRetryBaseDelay
	config = config.WithRetryBaseDelay(200 * time.Millisecond)
	if config.RetryBaseDelay != 200*time.Millisecond {
		t.Errorf("WithRetryBaseDelay failed: got %v", config.RetryBaseDelay)
	}

	// Test WithRetryMaxDelay
	config = config.WithRetryMaxDelay(60 * time.Second)
	if config.RetryMaxDelay != 60*time.Second {
		t.Errorf("WithRetryMaxDelay failed: got %v", config.RetryMaxDelay)
	}

	// Test WithRetryJitter
	config = config.WithRetryJitter(false)
	if config.RetryJitterEnabled {
		t.Errorf("WithRetryJitter failed: got %v", config.RetryJitterEnabled)
	}
}

func TestApicurioConfig_HasBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		expected bool
	}{
		{
			name:     "both username and password",
			username: testUsername,
			password: testPassword,
			expected: true,
		},
		{
			name:     "empty username",
			username: "",
			password: testPassword,
			expected: false,
		},
		{
			name:     "empty password",
			username: testUsername,
			password: "",
			expected: false,
		},
		{
			name:     "both empty",
			username: "",
			password: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewApicurioConfig().WithBasicAuth(tt.username, tt.password)
			result := config.HasBasicAuth()
			if result != tt.expected {
				t.Errorf("HasBasicAuth() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestApicurioConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		configFunc  func(*ApicurioConfig) *ApicurioConfig
		expectedErr bool
		errContains string
	}{
		{
			name:        "valid default config",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c },
			expectedErr: false,
		},
		{
			name:        "empty base URL",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithBaseURL("") },
			expectedErr: true,
			errContains: "BaseURL cannot be empty",
		},
		{
			name:        "negative timeout",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithTimeout(-1 * time.Second) },
			expectedErr: true,
			errContains: "timeout must be positive",
		},
		{
			name:        "zero timeout",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithTimeout(0) },
			expectedErr: true,
			errContains: "timeout must be positive",
		},
		{
			name:        "negative max retries",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithMaxRetries(-1) },
			expectedErr: true,
			errContains: "MaxRetries cannot be negative",
		},
		{
			name:        "negative schema cache size",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithSchemaCacheSize(-1) },
			expectedErr: true,
			errContains: "SchemaCacheSize cannot be negative",
		},
		{
			name:        "negative failed lookup cache size",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithFailedLookupCacheSize(-1) },
			expectedErr: true,
			errContains: "FailedLookupCacheSize cannot be negative",
		},
		{
			name:        "negative failed lookup cache TTL",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithFailedLookupCacheTTL(-1 * time.Second) },
			expectedErr: true,
			errContains: "FailedLookupCacheTTL cannot be negative",
		},
		{
			name:        "zero retry base delay",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithRetryBaseDelay(0) },
			expectedErr: true,
			errContains: "RetryBaseDelay must be positive",
		},
		{
			name:        "negative retry base delay",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithRetryBaseDelay(-1 * time.Second) },
			expectedErr: true,
			errContains: "RetryBaseDelay must be positive",
		},
		{
			name:        "zero retry max delay",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithRetryMaxDelay(0) },
			expectedErr: true,
			errContains: "RetryMaxDelay must be positive",
		},
		{
			name:        "negative retry max delay",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithRetryMaxDelay(-1 * time.Second) },
			expectedErr: true,
			errContains: "RetryMaxDelay must be positive",
		},
		{
			name: "base delay greater than max delay",
			configFunc: func(c *ApicurioConfig) *ApicurioConfig {
				return c.WithRetryBaseDelay(5 * time.Second).WithRetryMaxDelay(1 * time.Second)
			},
			expectedErr: true,
			errContains: "RetryBaseDelay (5s) cannot be greater than RetryMaxDelay (1s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.configFunc(NewApicurioConfig())
			err := config.Validate()

			if tt.expectedErr {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestApicurioConfig_String(t *testing.T) {
	tests := []struct {
		name        string
		configFunc  func(*ApicurioConfig) *ApicurioConfig
		contains    []string
		notContains []string
	}{
		{
			name:       "default config",
			configFunc: func(c *ApicurioConfig) *ApicurioConfig { return c },
			contains:   []string{"ApicurioConfig", defaultBaseURL, "30s", "MaxRetries: 3", "Auth: none"},
		},
		{
			name:        "config with basic auth",
			configFunc:  func(c *ApicurioConfig) *ApicurioConfig { return c.WithBasicAuth(testUsername, testPassword) },
			contains:    []string{"ApicurioConfig", "Auth: basic"},
			notContains: []string{"Auth: none"},
		},
		{
			name: "custom config",
			configFunc: func(c *ApicurioConfig) *ApicurioConfig {
				return c.WithBaseURL("https://custom.com").
					WithTimeout(45 * time.Second).
					WithMaxRetries(2)
			},
			contains: []string{"https://custom.com", "45s", "MaxRetries: 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.configFunc(NewApicurioConfig())
			str := config.String()

			for _, expected := range tt.contains {
				if !strings.Contains(str, expected) {
					t.Errorf("String() = %v, should contain %v", str, expected)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(str, notExpected) {
					t.Errorf("String() = %v, should not contain %v", str, notExpected)
				}
			}
		})
	}
}

func TestApicurioConfig_ChainedCalls(t *testing.T) {
	// Test that methods can be chained
	config := NewApicurioConfig().
		WithBaseURL("https://test.com").
		WithTimeout(45*time.Second).
		WithMaxRetries(2).
		WithBasicAuth("testuser", "testpass").
		WithSchemaCacheSize(500).
		WithFailedLookupCacheSize(50).
		WithFailedLookupCacheTTL(2 * time.Minute).
		WithRetryBaseDelay(200 * time.Millisecond).
		WithRetryMaxDelay(60 * time.Second).
		WithRetryJitter(false)

	// Verify all settings were applied
	if config.BaseURL != "https://test.com" {
		t.Errorf("Chained BaseURL = %v, want https://test.com", config.BaseURL)
	}

	if config.Timeout != 45*time.Second {
		t.Errorf("Chained Timeout = %v, want 45s", config.Timeout)
	}

	if config.MaxRetries != 2 {
		t.Errorf("Chained MaxRetries = %v, want 2", config.MaxRetries)
	}

	if !config.HasBasicAuth() {
		t.Errorf("Chained config should have basic auth")
	}

	if config.Username != "testuser" || config.Password != "testpass" {
		t.Errorf("Chained auth = %v:%v, want testuser:testpass", config.Username, config.Password)
	}

	if config.SchemaCacheSize != 500 {
		t.Errorf("Chained SchemaCacheSize = %v, want 500", config.SchemaCacheSize)
	}

	if config.FailedLookupCacheSize != 50 {
		t.Errorf("Chained FailedLookupCacheSize = %v, want 50", config.FailedLookupCacheSize)
	}

	if config.FailedLookupCacheTTL != 2*time.Minute {
		t.Errorf("Chained FailedLookupCacheTTL = %v, want 2m", config.FailedLookupCacheTTL)
	}

	if config.RetryBaseDelay != 200*time.Millisecond {
		t.Errorf("Chained RetryBaseDelay = %v, want 200ms", config.RetryBaseDelay)
	}

	if config.RetryMaxDelay != 60*time.Second {
		t.Errorf("Chained RetryMaxDelay = %v, want 60s", config.RetryMaxDelay)
	}

	if config.RetryJitterEnabled {
		t.Errorf("Chained RetryJitterEnabled = %v, want false", config.RetryJitterEnabled)
	}

	// Test that the config is still valid
	if err := config.Validate(); err != nil {
		t.Errorf("Chained config validation failed: %v", err)
	}
}
