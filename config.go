package avrocurio

import (
	"fmt"
	"time"
)

// ApicurioConfig represents configuration for Apicurio Schema Registry client.
type ApicurioConfig struct {
	// BaseURL is the base URL of the Apicurio Registry instance
	BaseURL string

	// Timeout is the HTTP request timeout
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts for failed requests
	MaxRetries int

	// Username and Password for basic authentication (optional)
	Username string
	Password string

	// SchemaCacheSize is the maximum number of schemas to cache (0 disables caching)
	SchemaCacheSize int

	// FailedLookupCacheSize is the maximum number of failed lookups to cache (0 disables caching)
	FailedLookupCacheSize int

	// FailedLookupCacheTTL is the TTL for failed lookup cache entries
	FailedLookupCacheTTL time.Duration

	// RetryBaseDelay is the base delay for exponential backoff (default: 100ms)
	RetryBaseDelay time.Duration

	// RetryMaxDelay is the maximum delay between retries (default: 30s)
	RetryMaxDelay time.Duration

	// RetryJitterEnabled controls whether jitter is added to retry delays (default: true)
	RetryJitterEnabled bool
}

// NewApicurioConfig creates a new ApicurioConfig with default values.
func NewApicurioConfig() *ApicurioConfig {
	return &ApicurioConfig{
		BaseURL:               "http://localhost:8080",
		Timeout:               30 * time.Second,
		MaxRetries:            3,
		SchemaCacheSize:       1000,
		FailedLookupCacheSize: 100,
		FailedLookupCacheTTL:  5 * time.Minute,
		RetryBaseDelay:        100 * time.Millisecond,
		RetryMaxDelay:         30 * time.Second,
		RetryJitterEnabled:    true,
	}
}

// WithBaseURL sets the base URL for the Apicurio Registry.
func (c *ApicurioConfig) WithBaseURL(baseURL string) *ApicurioConfig {
	c.BaseURL = baseURL
	return c
}

// WithTimeout sets the HTTP request timeout.
func (c *ApicurioConfig) WithTimeout(timeout time.Duration) *ApicurioConfig {
	c.Timeout = timeout
	return c
}

// WithMaxRetries sets the maximum number of retry attempts.
func (c *ApicurioConfig) WithMaxRetries(maxRetries int) *ApicurioConfig {
	c.MaxRetries = maxRetries
	return c
}

// WithBasicAuth sets basic authentication credentials.
func (c *ApicurioConfig) WithBasicAuth(username, password string) *ApicurioConfig {
	c.Username = username
	c.Password = password
	return c
}

// WithSchemaCacheSize sets the schema cache size.
func (c *ApicurioConfig) WithSchemaCacheSize(size int) *ApicurioConfig {
	c.SchemaCacheSize = size
	return c
}

// WithFailedLookupCacheSize sets the failed lookup cache size.
func (c *ApicurioConfig) WithFailedLookupCacheSize(size int) *ApicurioConfig {
	c.FailedLookupCacheSize = size
	return c
}

// WithFailedLookupCacheTTL sets the failed lookup cache TTL.
func (c *ApicurioConfig) WithFailedLookupCacheTTL(ttl time.Duration) *ApicurioConfig {
	c.FailedLookupCacheTTL = ttl
	return c
}

// WithRetryBaseDelay sets the base delay for exponential backoff.
func (c *ApicurioConfig) WithRetryBaseDelay(delay time.Duration) *ApicurioConfig {
	c.RetryBaseDelay = delay
	return c
}

// WithRetryMaxDelay sets the maximum delay between retries.
func (c *ApicurioConfig) WithRetryMaxDelay(delay time.Duration) *ApicurioConfig {
	c.RetryMaxDelay = delay
	return c
}

// WithRetryJitter enables or disables jitter in retry delays.
func (c *ApicurioConfig) WithRetryJitter(enabled bool) *ApicurioConfig {
	c.RetryJitterEnabled = enabled
	return c
}

// HasBasicAuth returns true if basic authentication is configured.
func (c *ApicurioConfig) HasBasicAuth() bool {
	return c.Username != "" && c.Password != ""
}

// Validate validates the configuration and returns an error if invalid.
func (c *ApicurioConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("BaseURL cannot be empty")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", c.Timeout)
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("MaxRetries cannot be negative, got %d", c.MaxRetries)
	}

	if c.SchemaCacheSize < 0 {
		return fmt.Errorf("SchemaCacheSize cannot be negative, got %d", c.SchemaCacheSize)
	}

	if c.FailedLookupCacheSize < 0 {
		return fmt.Errorf("FailedLookupCacheSize cannot be negative, got %d", c.FailedLookupCacheSize)
	}

	if c.FailedLookupCacheTTL < 0 {
		return fmt.Errorf("FailedLookupCacheTTL cannot be negative, got %v", c.FailedLookupCacheTTL)
	}

	if c.RetryBaseDelay <= 0 {
		return fmt.Errorf("RetryBaseDelay must be positive, got %v", c.RetryBaseDelay)
	}

	if c.RetryMaxDelay <= 0 {
		return fmt.Errorf("RetryMaxDelay must be positive, got %v", c.RetryMaxDelay)
	}

	if c.RetryBaseDelay > c.RetryMaxDelay {
		return fmt.Errorf("RetryBaseDelay (%v) cannot be greater than RetryMaxDelay (%v)", c.RetryBaseDelay, c.RetryMaxDelay)
	}

	return nil
}

// String returns a string representation of the configuration.
func (c *ApicurioConfig) String() string {
	auth := "none"
	if c.HasBasicAuth() {
		auth = "basic"
	}

	return fmt.Sprintf(
		"ApicurioConfig{BaseURL: %s, Timeout: %v, MaxRetries: %d, Auth: %s, SchemaCacheSize: %d, FailedLookupCacheSize: %d, FailedLookupCacheTTL: %v}",
		c.BaseURL,
		c.Timeout,
		c.MaxRetries,
		auth,
		c.SchemaCacheSize,
		c.FailedLookupCacheSize,
		c.FailedLookupCacheTTL,
	)
}
