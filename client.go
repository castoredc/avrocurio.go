package avrocurio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hamba/avro/v2"
	"github.com/jellydator/ttlcache/v3"
	"github.com/sethvargo/go-retry"
)

// CachedError represents a cached error from a failed schema lookup.
type CachedError struct {
	GlobalID   uint32
	StatusCode int
	Timestamp  time.Time
}

// ApicurioClient provides an HTTP client for interacting with Apicurio Schema Registry.
//
// This client provides methods to fetch schemas and manage schema registry operations
// using the Apicurio Registry v3 API with TTL caching for performance.
type ApicurioClient struct {
	config      *ApicurioConfig
	httpClient  *http.Client
	schemaCache *ttlcache.Cache[uint32, map[string]any]
	failedCache *ttlcache.Cache[uint32, *CachedError]
}

// ArtifactMetadata represents artifact metadata from the registry.
type ArtifactMetadata struct {
	ArtifactID string `json:"artifactId"`
	GroupID    string `json:"groupId"`
	Name       string `json:"name"`
	ID         string `json:"id"`
}

// SearchResults represents search results from the registry.
type SearchResults struct {
	Artifacts []ArtifactMetadata `json:"artifacts"`
}

// VersionMetadata represents version metadata for an artifact.
type VersionMetadata struct {
	GlobalID uint32 `json:"globalId"`
}

// VersionSearchResults represents the response from listing artifact versions.
type VersionSearchResults struct {
	Versions []VersionMetadata `json:"versions"`
}

// RegistrationResponse represents the response from schema registration.
type RegistrationResponse struct {
	Version  VersionMetadata  `json:"version"`
	Artifact ArtifactMetadata `json:"artifact"`
}

// NewApicurioClient creates a new ApicurioClient with the given configuration.
func NewApicurioClient(config *ApicurioConfig) (*ApicurioClient, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	client := &ApicurioClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	if config.SchemaCacheSize > 0 {
		client.schemaCache = ttlcache.New(
			ttlcache.WithCapacity[uint32, map[string]any](uint64(config.SchemaCacheSize)),
		)
		go client.schemaCache.Start()
	}

	if config.FailedLookupCacheSize > 0 {
		client.failedCache = ttlcache.New(
			ttlcache.WithCapacity[uint32, *CachedError](uint64(config.FailedLookupCacheSize)),
			ttlcache.WithTTL[uint32, *CachedError](config.FailedLookupCacheTTL),
		)
		go client.failedCache.Start()
	}

	return client, nil
}

// Close stops the cache cleanup routines.
func (c *ApicurioClient) Close() {
	if c.schemaCache != nil {
		c.schemaCache.Stop()
	}
	if c.failedCache != nil {
		c.failedCache.Stop()
	}
}

// checkFailedLookupCache checks if a global ID was previously failed and cached.
func (c *ApicurioClient) checkFailedLookupCache(globalID uint32) error {
	if c.failedCache == nil {
		return nil
	}

	if item := c.failedCache.Get(globalID); item != nil {
		cached := item.Value()
		elapsed := time.Since(cached.Timestamp)
		return fmt.Errorf("schema with global ID %d not found (cached %.1f seconds ago): %w",
			globalID, elapsed.Seconds(), ErrSchemaNotFound)
	}

	return nil
}

// cacheFailedLookup caches a failed schema lookup.
func (c *ApicurioClient) cacheFailedLookup(globalID uint32, statusCode int) {
	if c.failedCache == nil {
		return
	}

	cached := &CachedError{
		GlobalID:   globalID,
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}
	c.failedCache.Set(globalID, cached, ttlcache.DefaultTTL)
}

// makeRequest makes an HTTP request with proper authentication and error handling.
func (c *ApicurioClient) makeRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	fullURL := strings.TrimSuffix(c.config.BaseURL, "/") + url

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.config.HasBasicAuth() {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}
	backoff := retry.NewExponential(c.config.RetryBaseDelay)
	if c.config.RetryJitterEnabled {
		backoff = retry.WithJitter(c.config.RetryBaseDelay/4, backoff)
	}
	if c.config.RetryMaxDelay > 0 {
		backoff = retry.WithMaxDuration(c.config.RetryMaxDelay, backoff)
	}

	var resp *http.Response
	var lastErr error

	// Store original body for retries
	var originalBody []byte
	if body != nil {
		originalBody, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	err = retry.Do(ctx, retry.WithMaxRetries(uint64(c.config.MaxRetries), backoff), func(ctx context.Context) error {
		var requestBody io.Reader
		if originalBody != nil {
			requestBody = bytes.NewReader(originalBody)
		}

		// Create request for this attempt
		attemptReq, err := http.NewRequestWithContext(ctx, method, fullURL, requestBody)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to create request: %w", err))
		}

		// Copy headers from original request
		attemptReq.Header = req.Header.Clone()

		// Perform the request
		resp, err = c.httpClient.Do(attemptReq)
		if err != nil {
			// Network errors are retryable
			lastErr = err
			return retry.RetryableError(fmt.Errorf("network error: %w", err))
		}

		// Check if we should retry based on status code
		if c.shouldRetryStatusCode(resp.StatusCode) {
			// Close the response body for failed attempts
			resp.Body.Close() //nolint:errcheck
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			return retry.RetryableError(lastErr)
		}

		// Success or non-retryable error
		return nil
	})
	if err != nil {
		// If the error is a context error, preserve it
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if lastErr != nil {
			return nil, fmt.Errorf("request failed after retries: %w", lastErr)
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// shouldRetryStatusCode determines if a request should be retried based on status code
func (c *ApicurioClient) shouldRetryStatusCode(statusCode int) bool {
	// Retry on server errors (5xx) and some specific client errors
	switch {
	case statusCode >= 500: // 5xx server errors
		return true
	case statusCode == 408: // Request Timeout
		return true
	case statusCode == 429: // Too Many Requests
		return true
	default:
		return false
	}
}

// GetSchemaByGlobalID fetches schema by its global ID with caching.
func (c *ApicurioClient) GetSchemaByGlobalID(ctx context.Context, globalID uint32) (map[string]any, error) {
	// Check cache first
	if c.schemaCache != nil {
		if item := c.schemaCache.Get(globalID); item != nil {
			return item.Value(), nil
		}
	}

	// Check failed lookup cache
	if err := c.checkFailedLookupCache(globalID); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("/apis/registry/v3/ids/globalIds/%d", globalID)
	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		c.cacheFailedLookup(globalID, resp.StatusCode)
		return nil, fmt.Errorf("schema with global ID %d: %w", globalID, ErrSchemaNotFound)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(body, &schema); err != nil {
		return nil, fmt.Errorf("invalid JSON schema content: %w", err)
	}

	// Cache the schema
	if c.schemaCache != nil {
		c.schemaCache.Set(globalID, schema, ttlcache.NoTTL)
	}

	return schema, nil
}

// GetLatestSchema gets the latest version of a schema by group and artifact ID.
func (c *ApicurioClient) GetLatestSchema(ctx context.Context, groupID, artifactID string) (uint32, map[string]any, error) {
	url := fmt.Sprintf("/apis/registry/v3/groups/%s/artifacts/%s/versions/branch=latest", groupID, artifactID)
	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil, fmt.Errorf("schema %s/%s: %w", groupID, artifactID, ErrSchemaNotFound)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var versionMetadata VersionMetadata
	if err := json.NewDecoder(resp.Body).Decode(&versionMetadata); err != nil {
		return 0, nil, fmt.Errorf("failed to decode version metadata: %w", err)
	}

	globalID := versionMetadata.GlobalID
	if globalID == 0 {
		return 0, nil, fmt.Errorf("no global ID found in metadata for %s/%s", groupID, artifactID)
	}

	// Fetch the actual schema content
	schema, err := c.GetSchemaByGlobalID(ctx, globalID)
	if err != nil {
		return 0, nil, err
	}

	return globalID, schema, nil
}

// SearchArtifacts searches for artifacts in the registry.
func (c *ApicurioClient) SearchArtifacts(ctx context.Context, name, artifactType string) ([]ArtifactMetadata, error) {
	url := "/apis/registry/v3/search/artifacts"

	// Build query parameters
	params := make([]string, 0)
	if artifactType != "" {
		params = append(params, "artifactType="+artifactType)
	}
	if name != "" {
		params = append(params, "name="+name)
	}

	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
	}

	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var searchResults SearchResults
	if err := json.NewDecoder(resp.Body).Decode(&searchResults); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return searchResults.Artifacts, nil
}

// RegisterSchema registers a new schema in the registry and returns its global ID.
func (c *ApicurioClient) RegisterSchema(ctx context.Context, group, artifactName, schemaContent string) (uint32, error) {
	url := fmt.Sprintf("/apis/registry/v3/groups/%s/artifacts", group)

	artifactData := map[string]any{
		"artifactId":   artifactName,
		"artifactType": "AVRO",
		"firstVersion": map[string]any{
			"content": map[string]any{ //nolint:goconst
				"content":     schemaContent,
				"contentType": "application/json",
			},
		},
	}

	jsonData, err := json.Marshal(artifactData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal artifact data: %w", err)
	}

	// Add query parameter for idempotent behavior
	url += "?ifExists=FIND_OR_CREATE_VERSION"

	resp, err := c.makeRequest(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var registrationResponse RegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&registrationResponse); err != nil {
		return 0, fmt.Errorf("failed to decode registration response: %w", err)
	}

	globalID := registrationResponse.Version.GlobalID
	if globalID == 0 {
		return 0, fmt.Errorf("no global ID returned for registered schema %s/%s", group, artifactName)
	}

	return globalID, nil
}

// RegisterSchemaVersion registers a new version of an existing schema.
func (c *ApicurioClient) RegisterSchemaVersion(ctx context.Context, groupID, artifactID, schemaContent string) (uint32, error) {
	url := fmt.Sprintf("/apis/registry/v3/groups/%s/artifacts/%s/versions", groupID, artifactID)

	versionData := map[string]any{
		"content": map[string]any{
			"content":     schemaContent,
			"contentType": "application/json",
		},
	}

	jsonData, err := json.Marshal(versionData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal version data: %w", err)
	}

	resp, err := c.makeRequest(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var versionMetadata VersionMetadata
	if err := json.NewDecoder(resp.Body).Decode(&versionMetadata); err != nil {
		return 0, fmt.Errorf("failed to decode version metadata: %w", err)
	}

	globalID := versionMetadata.GlobalID
	if globalID == 0 {
		return 0, fmt.Errorf("no global ID returned for schema version %s/%s", groupID, artifactID)
	}

	return globalID, nil
}

// CheckArtifactExists checks if an artifact exists in the registry.
func (c *ApicurioClient) CheckArtifactExists(ctx context.Context, groupID, artifactID string) (bool, error) {
	url := fmt.Sprintf("/apis/registry/v3/groups/%s/artifacts/%s", groupID, artifactID)
	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close() //nolint:errcheck

	return resp.StatusCode == http.StatusOK, nil
}

// FindArtifactByContent finds an artifact by searching the registry with schema content.
// This method identifies which artifact contains a schema matching the provided content,
// but does not guarantee which version of that artifact. Use FindExactSchemaVersionByContent
// to get the specific version (global ID) that matches the content exactly.
func (c *ApicurioClient) FindArtifactByContent(ctx context.Context, schemaContent string, groupID *string) (string, string, error) {
	url := "/apis/registry/v3/search/artifacts"

	// Build query parameters
	params := []string{
		"canonical=true",
		"artifactType=AVRO",
		"limit=10",
	}

	if groupID != nil && *groupID != "" {
		params = append(params, "groupId="+*groupID)
	}

	url += "?" + strings.Join(params, "&")

	resp, err := c.makeRequest(ctx, "POST", url, strings.NewReader(schemaContent))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var searchResults SearchResults
	if err := json.NewDecoder(resp.Body).Decode(&searchResults); err != nil {
		return "", "", fmt.Errorf("failed to decode search results: %w", err)
	}

	if len(searchResults.Artifacts) == 0 {
		return "", "", nil // No match found
	}

	first := searchResults.Artifacts[0]
	artifactID := first.ArtifactID
	if artifactID == "" {
		artifactID = first.ID
		if artifactID == "" {
			artifactID = first.Name
			if artifactID == "" {
				artifactID = "unknown"
			}
		}
	}

	groupIDResult := first.GroupID
	if groupIDResult == "" {
		groupIDResult = "default"
	}

	return groupIDResult, artifactID, nil
}

// FindExactSchemaVersionByContent finds the exact schema version (global ID) that matches
// the provided schema content within a specific artifact. This differs from FindArtifactByContent
// which only identifies the artifact but not the specific version. This method prevents race
// conditions by ensuring we get the precise schema version that corresponds to the content,
// rather than just the latest version of the artifact.
func (c *ApicurioClient) FindExactSchemaVersionByContent(ctx context.Context, groupID, artifactID, schemaContent string) (uint32, error) {
	// Get all versions of the artifact
	url := fmt.Sprintf("/apis/registry/v3/groups/%s/artifacts/%s/versions", groupID, artifactID)
	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var versions VersionSearchResults
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return 0, fmt.Errorf("failed to decode versions: %w", err)
	}

	// Find the version with matching content
	for _, version := range versions.Versions {
		if version.GlobalID == 0 {
			continue
		}

		// Get the schema content for this version
		versionSchema, err := c.GetSchemaByGlobalID(ctx, version.GlobalID)
		if err != nil {
			continue // Skip this version if we can't get the schema
		}

		// Convert to JSON string for comparison
		versionJSON, err := json.Marshal(versionSchema)
		if err != nil {
			continue
		}

		// Parse both schemas to normalize them for comparison
		providedSchema, err := avro.Parse(schemaContent)
		if err != nil {
			continue
		}

		versionAvroSchema, err := avro.Parse(string(versionJSON))
		if err != nil {
			continue
		}

		// Compare canonical representations
		if providedSchema.String() == versionAvroSchema.String() {
			return version.GlobalID, nil
		}
	}

	return 0, fmt.Errorf("no schema version found with matching content for %s/%s", groupID, artifactID)
}

// ClearCache clears all cached schemas and failed lookups.
func (c *ApicurioClient) ClearCache() {
	if c.schemaCache != nil {
		c.schemaCache.DeleteAll()
	}
	if c.failedCache != nil {
		c.failedCache.DeleteAll()
	}
}

// GetCacheStats returns cache statistics.
func (c *ApicurioClient) GetCacheStats() map[string]any {
	stats := make(map[string]any)

	if c.schemaCache != nil {
		stats["schema_cache_size"] = c.schemaCache.Len()
		stats["schema_cache_max_size"] = c.config.SchemaCacheSize
	} else {
		stats["schema_cache_size"] = 0
		stats["schema_cache_max_size"] = 0
	}

	if c.failedCache != nil {
		stats["failed_lookup_cache_size"] = c.failedCache.Len()
		stats["failed_lookup_cache_max_size"] = c.config.FailedLookupCacheSize
		stats["failed_lookup_cache_ttl"] = c.config.FailedLookupCacheTTL.Seconds()
	} else {
		stats["failed_lookup_cache_size"] = 0
		stats["failed_lookup_cache_max_size"] = 0
		stats["failed_lookup_cache_ttl"] = 0
	}

	return stats
}
