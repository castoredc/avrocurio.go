//go:build integration

package avrocurio

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hamba/avro/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test event models equivalent to Python dataclass models

// IntegrationTestEvent represents a simple test schema for integration tests
type IntegrationTestEvent struct {
	ID        string `json:"id" avro:"id"`
	Timestamp int64  `json:"timestamp" avro:"timestamp"`
	Message   string `json:"message" avro:"message"`
}

// Schema evolution test models

// StatusV1 represents version 1 of status enum
type StatusV1 int

const (
	StatusV1Unknown StatusV1 = iota
	StatusV1Active
	StatusV1Inactive
)

func (s StatusV1) String() string {
	switch s {
	case StatusV1Active:
		return "ACTIVE"
	case StatusV1Inactive:
		return "INACTIVE"
	default:
		return "UNKNOWN"
	}
}

// StatusV2 represents version 2 of status enum (adds new member)
type StatusV2 int

const (
	StatusV2Unknown StatusV2 = iota
	StatusV2Active
	StatusV2Inactive
	StatusV2Pending
)

func (s StatusV2) String() string {
	switch s {
	case StatusV2Active:
		return "ACTIVE"
	case StatusV2Inactive:
		return "INACTIVE"
	case StatusV2Pending:
		return "PENDING"
	default:
		return "UNKNOWN"
	}
}

// StatusV3 represents version 3 of status enum (removes member)
type StatusV3 int

const (
	StatusV3Unknown StatusV3 = iota
	StatusV3Active
	StatusV3Inactive
)

func (s StatusV3) String() string {
	switch s {
	case StatusV3Active:
		return "ACTIVE"
	case StatusV3Inactive:
		return "INACTIVE"
	default:
		return "UNKNOWN"
	}
}

// UserV1 represents version 1 - Base user schema
type UserV1 struct {
	Name   string   `json:"name" avro:"name"`
	Email  string   `json:"email" avro:"email"`
	Status StatusV1 `json:"status" avro:"status"`
}

// UserV2 represents version 2 - Backward compatible (adds optional fields)
type UserV2 struct {
	Name     string   `json:"name" avro:"name"`
	Email    string   `json:"email" avro:"email"`
	Status   StatusV2 `json:"status" avro:"status"`
	Age      int      `json:"age" avro:"age"`
	IsActive bool     `json:"is_active" avro:"is_active"`
}

// UserV3 represents version 3 - Type promotion and enum evolution
type UserV3 struct {
	Name     string   `json:"name" avro:"name"`
	Email    string   `json:"email" avro:"email"`
	Status   StatusV3 `json:"status" avro:"status"`
	Age      int      `json:"age" avro:"age"`
	IsActive bool     `json:"is_active" avro:"is_active"`
	Score    float64  `json:"score" avro:"score"`
}

// Registration test models
type RegistrationTestEvent struct {
	EventID   string `json:"event_id" avro:"event_id"`
	Timestamp int64  `json:"timestamp" avro:"timestamp"`
	Data      string `json:"data" avro:"data"`
	Version   int    `json:"version" avro:"version"`
}

type UpdatedRegistrationTestEvent struct {
	EventID   string  `json:"event_id" avro:"event_id"`
	Timestamp int64   `json:"timestamp" avro:"timestamp"`
	Data      string  `json:"data" avro:"data"`
	Version   int     `json:"version" avro:"version"`
	NewField  *string `json:"new_field,omitempty" avro:"new_field"`
}

// Test helper functions

// createTestConfig creates test configuration for Apicurio
func createTestConfig() *ApicurioConfig {
	baseURL := os.Getenv("APICURIO_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return NewApicurioConfig().WithBaseURL(baseURL).WithTimeout(30 * time.Second)
}

// createTestGroupID creates unique group ID for test isolation
func createTestGroupID() string {
	return fmt.Sprintf("test-group-%s", uuid.New().String()[:8])
}

// createTestArtifactID creates unique artifact ID for test isolation
func createTestArtifactID() string {
	return fmt.Sprintf("test-event-%s", uuid.New().String()[:8])
}

// Schema definitions for test models

func getIntegrationTestEventSchema() string {
	return `{
		"type": "record",
		"name": "IntegrationTestEvent",
		"fields": [
			{"name": "id", "type": "string"},
			{"name": "timestamp", "type": "long"},
			{"name": "message", "type": "string"}
		]
	}`
}

func getUserV1Schema() string {
	return `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "name", "type": "string"},
			{"name": "email", "type": "string"},
			{
				"name": "status",
				"type": {
					"type": "enum",
					"name": "Status",
					"symbols": ["UNKNOWN", "ACTIVE", "INACTIVE"],
					"default": "UNKNOWN"
				},
				"default": "UNKNOWN"
			}
		]
	}`
}

func getUserV2Schema() string {
	return `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "name", "type": "string"},
			{"name": "email", "type": "string"},
			{
				"name": "status",
				"type": {
					"type": "enum",
					"name": "Status",
					"symbols": ["UNKNOWN", "ACTIVE", "INACTIVE", "PENDING"],
					"default": "UNKNOWN"
				},
				"default": "UNKNOWN"
			},
			{"name": "age", "type": "int", "default": 25},
			{"name": "is_active", "type": "boolean", "default": true}
		]
	}`
}

func getUserV3Schema() string {
	return `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "name", "type": "string"},
			{"name": "email", "type": "string"},
			{
				"name": "status",
				"type": {
					"type": "enum",
					"name": "Status",
					"symbols": ["UNKNOWN", "ACTIVE", "INACTIVE"],
					"default": "UNKNOWN"
				},
				"default": "UNKNOWN"
			},
			{"name": "age", "type": "int", "default": 25},
			{"name": "is_active", "type": "boolean", "default": true},
			{"name": "score", "type": "double", "default": 0.0}
		]
	}`
}

func getRegistrationTestEventSchema() string {
	return `{
		"type": "record",
		"name": "RegistrationTestEvent",
		"fields": [
			{"name": "event_id", "type": "string"},
			{"name": "timestamp", "type": "long"},
			{"name": "data", "type": "string"},
			{"name": "version", "type": "int", "default": 1}
		]
	}`
}

func getUpdatedRegistrationTestEventSchema() string {
	return `{
		"type": "record",
		"name": "RegistrationTestEvent",
		"fields": [
			{"name": "event_id", "type": "string"},
			{"name": "timestamp", "type": "long"},
			{"name": "data", "type": "string"},
			{"name": "version", "type": "int", "default": 2},
			{"name": "new_field", "type": ["null", "string"], "default": null}
		]
	}`
}

// END-TO-END INTEGRATION TESTS
// Port of test_end_to_end.py

func TestBasicRoundTripWithRegisteredSchema(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)
	groupID := createTestGroupID()
	artifactID := createTestArtifactID()

	// Create test event
	testEvent := IntegrationTestEvent{
		ID:        artifactID,
		Timestamp: time.Now().Unix(),
		Message:   "Hello from AvroCurio integration test!",
	}

	// Register schema
	schemaContent := getIntegrationTestEventSchema()
	globalID, err := client.RegisterSchema(ctx, groupID, artifactID, schemaContent)
	require.NoError(t, err)

	// Parse schema for serialization
	_, err = avro.Parse(schemaContent)
	require.NoError(t, err)

	// Test serialization with registered schema
	serialized, err := serializer.SerializeWithSchemaID(ctx, testEvent, globalID)
	require.NoError(t, err)

	// Verify wire format structure
	require.GreaterOrEqual(t, len(serialized), 5) // magic byte + 4 bytes schema ID + payload
	assert.Equal(t, byte(0x0), serialized[0])     // Magic byte

	// Test deserialization
	var deserialized IntegrationTestEvent
	err = serializer.Deserialize(ctx, serialized, &deserialized)
	require.NoError(t, err)

	// Verify round trip integrity
	assert.Equal(t, testEvent.ID, deserialized.ID)
	assert.Equal(t, testEvent.Timestamp, deserialized.Timestamp)
	assert.Equal(t, testEvent.Message, deserialized.Message)
}

func TestSchemaRetrievalBySubject(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-retrieval", createTestArtifactID())

	// Register schema
	schemaContent := getIntegrationTestEventSchema()
	registeredGlobalID, err := client.RegisterSchema(ctx, groupID, artifactID, schemaContent)
	require.NoError(t, err)

	// Test retrieving latest schema for the artifact we just registered
	globalID, schema, err := client.GetLatestSchema(ctx, groupID, artifactID)
	require.NoError(t, err)

	// Verify we got valid results
	assert.IsType(t, uint32(0), globalID)
	assert.Greater(t, globalID, uint32(0))
	assert.IsType(t, map[string]interface{}{}, schema)
	assert.Equal(t, registeredGlobalID, globalID)

	// Verify we can also get schema by global ID
	retrievedSchema, err := client.GetSchemaByGlobalID(ctx, globalID)
	require.NoError(t, err)
	assert.NotNil(t, retrievedSchema)
}

func TestWireFormatStructure(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-wire-format", createTestArtifactID())

	// Register schema
	schemaContent := getIntegrationTestEventSchema()
	globalID, err := client.RegisterSchema(ctx, groupID, artifactID, schemaContent)
	require.NoError(t, err)

	// Create dummy payload (we're just testing wire format structure)
	dummyPayload := []byte("dummy avro data")

	// Test wire format encoding
	wireFormat := &ConfluentWireFormat{}
	wireMessage := wireFormat.Encode(globalID, dummyPayload)

	// Verify wire format structure
	require.GreaterOrEqual(t, len(wireMessage), 5) // magic byte + 4 bytes schema ID + payload
	assert.Equal(t, byte(0x0), wireMessage[0])     // Magic byte

	// Test wire format decoding
	decodedSchemaID, decodedPayload, err := wireFormat.Decode(wireMessage)
	require.NoError(t, err)

	assert.Equal(t, globalID, decodedSchemaID)
	assert.Equal(t, dummyPayload, decodedPayload)
}

func TestSchemaNotFoundError(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	// Try to get a schema with an ID that definitely doesn't exist
	nonExistentID := uint32(999999999)

	_, err = client.GetSchemaByGlobalID(ctx, nonExistentID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSchemaNotFound)
}

func TestRegistryConnectivity(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-connectivity", createTestArtifactID())

	// Test that we can search artifacts
	artifacts, err := client.SearchArtifacts(ctx, "", "")
	require.NoError(t, err)
	assert.IsType(t, []ArtifactMetadata{}, artifacts)

	// Test that we can register a schema to verify write access
	schemaContent := getIntegrationTestEventSchema()
	globalID, err := client.RegisterSchema(ctx, groupID, artifactID, schemaContent)
	require.NoError(t, err)
	assert.NotZero(t, globalID)
	assert.IsType(t, uint32(0), globalID)

	// Verify we can retrieve the schema we just registered
	retrievedGlobalID, schema, err := client.GetLatestSchema(ctx, groupID, artifactID)
	require.NoError(t, err)
	assert.Equal(t, globalID, retrievedGlobalID)
	assert.NotNil(t, schema)
}

func TestSerializeAutomaticLookupRegistryWide(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)
	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-serialize-auto", createTestArtifactID())

	// Create test event
	testEvent := IntegrationTestEvent{
		ID:        artifactID,
		Timestamp: time.Now().Unix(),
		Message:   "Serialize automatic lookup test!",
	}

	// Register schema
	schemaContent := getIntegrationTestEventSchema()
	_, err = client.RegisterSchema(ctx, groupID, artifactID, schemaContent)
	require.NoError(t, err)

	// Parse schema for serialization
	schema, err := avro.Parse(schemaContent)
	require.NoError(t, err)

	// Test serialize with automatic lookup (no schema_id provided)
	serialized, err := serializer.Serialize(ctx, testEvent, schema)
	require.NoError(t, err)

	// Test deserialization to verify round trip
	var deserialized IntegrationTestEvent
	err = serializer.Deserialize(ctx, serialized, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, testEvent.ID, deserialized.ID)
	assert.Equal(t, testEvent.Timestamp, deserialized.Timestamp)
	assert.Equal(t, testEvent.Message, deserialized.Message)
}

func TestSerializeExplicitSchema(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)
	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-serialize-schema-id", createTestArtifactID())

	// Create test event
	testEvent := IntegrationTestEvent{
		ID:        artifactID,
		Timestamp: time.Now().Unix(),
		Message:   "Serialize with explicit schema ID",
	}

	// Register schema
	schemaContent := getIntegrationTestEventSchema()
	globalID, err := client.RegisterSchema(ctx, groupID, artifactID, schemaContent)
	require.NoError(t, err)

	// Parse schema for serialization
	_, err = avro.Parse(schemaContent)
	require.NoError(t, err)

	// Test serialize with explicit schema ID
	serialized, err := serializer.SerializeWithSchemaID(ctx, testEvent, globalID)
	require.NoError(t, err)

	// Test deserialization to verify round trip
	var deserialized IntegrationTestEvent
	err = serializer.Deserialize(ctx, serialized, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, testEvent.ID, deserialized.ID)
	assert.Equal(t, testEvent.Timestamp, deserialized.Timestamp)
	assert.Equal(t, testEvent.Message, deserialized.Message)
}

func TestSerializeAutomaticLookupNoMatchIntegration(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)

	// Create a unique schema that won't exist in registry
	uniqueSchema := `{
		"type": "record",
		"name": "VeryUniqueTestEvent",
		"fields": [
			{"name": "extremely_unique_field", "type": "string"},
			{"name": "another_super_specific_field", "type": "int"},
			{"name": "totally_never_registered_field", "type": "boolean"}
		]
	}`

	uniqueEvent := map[string]interface{}{
		"extremely_unique_field":         "serialize-unique",
		"another_super_specific_field":   999,
		"totally_never_registered_field": false,
	}

	// Parse schema for serialization
	schema, err := avro.Parse(uniqueSchema)
	require.NoError(t, err)

	// Test that ErrSchemaNotMatched is returned for serialize() automatic lookup
	_, err = serializer.Serialize(ctx, uniqueEvent, schema)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSchemaNotMatched)
}

func TestRegisterSchemaEndToEndWorkflow(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)
	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-e2e-workflow", createTestArtifactID())

	// Create test event
	testEvent := IntegrationTestEvent{
		ID:        fmt.Sprintf("%s-e2e", artifactID),
		Timestamp: time.Now().Unix(),
		Message:   "End-to-end register_schema workflow test!",
	}

	// Parse schema for registration
	schemaContent := getIntegrationTestEventSchema()
	schema, err := avro.Parse(schemaContent)
	require.NoError(t, err)

	// Register schema using serializer
	globalID, err := serializer.Client().RegisterSchema(ctx, groupID, artifactID, schema.String())
	require.NoError(t, err)

	// Verify registration succeeded
	assert.IsType(t, uint32(0), globalID)
	assert.Greater(t, globalID, uint32(0))

	// Test serialization with returned global ID
	serialized, err := serializer.SerializeWithSchemaID(ctx, testEvent, globalID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(serialized), 5)
	assert.Equal(t, byte(0x0), serialized[0])

	// Test deserialization
	var deserialized IntegrationTestEvent
	err = serializer.Deserialize(ctx, serialized, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, testEvent.ID, deserialized.ID)
	assert.Equal(t, testEvent.Timestamp, deserialized.Timestamp)
	assert.Equal(t, testEvent.Message, deserialized.Message)

	// Test that the registered schema can now be found via automatic lookup
	serializedAuto, err := serializer.Serialize(ctx, testEvent, schema)
	require.NoError(t, err)

	var deserializedAuto IntegrationTestEvent
	err = serializer.Deserialize(ctx, serializedAuto, &deserializedAuto)
	require.NoError(t, err)

	assert.Equal(t, deserializedAuto.ID, testEvent.ID)
	assert.Equal(t, deserializedAuto.Timestamp, testEvent.Timestamp)
	assert.Equal(t, deserializedAuto.Message, testEvent.Message)
}

func TestDeserializeToRawDict(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)
	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-raw-dict", createTestArtifactID())

	// Create test event
	testEvent := IntegrationTestEvent{
		ID:        fmt.Sprintf("%s-raw-dict", artifactID),
		Timestamp: time.Now().Unix(),
		Message:   "Raw dictionary deserialization integration test!",
	}

	// Register schema
	schemaContent := getIntegrationTestEventSchema()
	globalID, err := client.RegisterSchema(ctx, groupID, artifactID, schemaContent)
	require.NoError(t, err)

	// Parse schema for serialization
	_, err = avro.Parse(schemaContent)
	require.NoError(t, err)

	// Serialize
	serialized, err := serializer.SerializeWithSchemaID(ctx, testEvent, globalID)
	require.NoError(t, err)

	// Deserialize to raw interface{}
	deserialized, err := serializer.DeserializeToInterface(ctx, serialized)
	require.NoError(t, err)

	// Verify result is a map with expected data
	result, ok := deserialized.(map[string]interface{})
	require.True(t, ok, "Deserialized data should be a map")
	assert.Equal(t, testEvent.ID, result["id"])
	assert.Equal(t, testEvent.Timestamp, result["timestamp"])
	assert.Equal(t, testEvent.Message, result["message"])
}

// SCHEMA REGISTRATION TESTS
// Port of test_schema_registration.py

func TestRegisterSchemaSuccess(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)
	groupID := createTestGroupID()
	artifactID := createTestArtifactID()

	// Create test event (not used directly, just for demonstration)
	_ = RegistrationTestEvent{
		EventID:   artifactID,
		Timestamp: time.Now().Unix(),
		Data:      "Registration test data",
		Version:   1,
	}

	// Parse schema for registration
	schemaContent := getRegistrationTestEventSchema()
	schema, err := avro.Parse(schemaContent)
	require.NoError(t, err)

	// Register the schema
	globalID, err := serializer.Client().RegisterSchema(ctx, groupID, artifactID, schema.String())
	require.NoError(t, err)

	// Verify we got a valid global ID
	assert.IsType(t, uint32(0), globalID)
	assert.Greater(t, globalID, uint32(0))

	// Verify we can retrieve the schema using the returned global ID
	schemaMap, err := client.GetSchemaByGlobalID(ctx, globalID)
	require.NoError(t, err)
	assert.NotNil(t, schemaMap)
	assert.IsType(t, map[string]interface{}{}, schemaMap)
}

func TestRegisterSchemaIdempotent(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)
	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-duplicate", createTestArtifactID())

	// Parse schema for registration
	schemaContent := getRegistrationTestEventSchema()
	schema, err := avro.Parse(schemaContent)
	require.NoError(t, err)

	// Register the schema for the first time
	firstGlobalID, err := serializer.Client().RegisterSchema(ctx, groupID, artifactID, schema.String())
	require.NoError(t, err)

	// Register the same schema again
	secondGlobalID, err := serializer.Client().RegisterSchema(ctx, groupID, artifactID, schema.String())
	require.NoError(t, err)

	// Both registrations should succeed and return the same global ID
	// since the schema content is identical (idempotent behavior)
	assert.IsType(t, uint32(0), firstGlobalID)
	assert.IsType(t, uint32(0), secondGlobalID)
	assert.Greater(t, firstGlobalID, uint32(0))
	assert.Greater(t, secondGlobalID, uint32(0))
	assert.Equal(t, firstGlobalID, secondGlobalID)
}

func TestRegisterSchemaVersionUpdate(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)
	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-evolution", createTestArtifactID())

	// Register initial schema version
	schemaV1Content := getRegistrationTestEventSchema()
	schemaV1, err := avro.Parse(schemaV1Content)
	require.NoError(t, err)

	firstGlobalID, err := serializer.Client().RegisterSchema(ctx, groupID, artifactID, schemaV1.String())
	require.NoError(t, err)

	// Register evolved schema version
	schemaV2Content := getUpdatedRegistrationTestEventSchema()
	schemaV2, err := avro.Parse(schemaV2Content)
	require.NoError(t, err)

	secondGlobalID, err := serializer.Client().RegisterSchema(ctx, groupID, artifactID, schemaV2.String())
	require.NoError(t, err)

	// Both should succeed and have different global IDs
	assert.IsType(t, uint32(0), firstGlobalID)
	assert.IsType(t, uint32(0), secondGlobalID)
	assert.NotEqual(t, firstGlobalID, secondGlobalID)

	// Verify both schemas can be retrieved
	firstSchema, err := client.GetSchemaByGlobalID(ctx, firstGlobalID)
	require.NoError(t, err)
	secondSchema, err := client.GetSchemaByGlobalID(ctx, secondGlobalID)
	require.NoError(t, err)

	assert.NotEqual(t, firstSchema, secondSchema)

	// Test serialization/deserialization with both versions
	// Create test events for each version
	initialEvent := RegistrationTestEvent{
		EventID:   artifactID,
		Timestamp: time.Now().Unix(),
		Data:      "Evolution test - v1",
		Version:   1,
	}

	evolvedEvent := UpdatedRegistrationTestEvent{
		EventID:   artifactID,
		Timestamp: time.Now().Unix(),
		Data:      "Evolution test - v2",
		Version:   2,
		NewField:  stringPtr("New field value"),
	}

	// Serialize v1 data with v1 schema
	v1Serialized, err := serializer.SerializeWithSchemaID(ctx, initialEvent, firstGlobalID)
	require.NoError(t, err)

	var v1Deserialized RegistrationTestEvent
	err = serializer.Deserialize(ctx, v1Serialized, &v1Deserialized)
	require.NoError(t, err)
	assert.Equal(t, initialEvent.EventID, v1Deserialized.EventID)
	assert.Equal(t, initialEvent.Version, v1Deserialized.Version)

	// Serialize v2 data with v2 schema
	v2Serialized, err := serializer.SerializeWithSchemaID(ctx, evolvedEvent, secondGlobalID)
	require.NoError(t, err)

	var v2Deserialized UpdatedRegistrationTestEvent
	err = serializer.Deserialize(ctx, v2Serialized, &v2Deserialized)
	require.NoError(t, err)
	assert.Equal(t, evolvedEvent.EventID, v2Deserialized.EventID)
	assert.Equal(t, evolvedEvent.Version, v2Deserialized.Version)
	assert.Equal(t, "New field value", *v2Deserialized.NewField)

	// Test backwards compatibility: v1 data with v2 schema
	// Data serialized with v1 should be readable with v2 (new field has default)
	var v1DataV2Schema UpdatedRegistrationTestEvent
	err = serializer.Deserialize(ctx, v1Serialized, &v1DataV2Schema)
	require.NoError(t, err)
	assert.Equal(t, initialEvent.EventID, v1DataV2Schema.EventID)
	assert.Equal(t, initialEvent.Version, v1DataV2Schema.Version) // Original version value
	assert.Nil(t, v1DataV2Schema.NewField)                        // Default value for missing field
}

func TestRegisterThenSerializeWithReturnedID(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)
	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-serialize", createTestArtifactID())

	testEvent := RegistrationTestEvent{
		EventID:   artifactID,
		Timestamp: time.Now().Unix(),
		Data:      "Register then serialize test",
		Version:   1,
	}

	// Parse schema for registration
	schemaContent := getRegistrationTestEventSchema()
	schema, err := avro.Parse(schemaContent)
	require.NoError(t, err)

	// Register the schema
	globalID, err := serializer.Client().RegisterSchema(ctx, groupID, artifactID, schema.String())
	require.NoError(t, err)

	// Use the returned global ID for serialization
	serialized, err := serializer.SerializeWithSchemaID(ctx, testEvent, globalID)
	require.NoError(t, err)

	// Verify serialization worked
	assert.IsType(t, []byte{}, serialized)
	require.GreaterOrEqual(t, len(serialized), 5) // magic byte + 4 bytes schema ID + payload
	assert.Equal(t, byte(0x0), serialized[0])     // Magic byte

	// Test deserialization round trip
	var deserialized RegistrationTestEvent
	err = serializer.Deserialize(ctx, serialized, &deserialized)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, testEvent.EventID, deserialized.EventID)
	assert.Equal(t, testEvent.Timestamp, deserialized.Timestamp)
	assert.Equal(t, testEvent.Data, deserialized.Data)
	assert.Equal(t, testEvent.Version, deserialized.Version)
}

func TestRegisterThenAutomaticLookup(t *testing.T) {
	ctx := context.Background()
	config := createTestConfig()
	client, err := NewApicurioClient(config)
	require.NoError(t, err)
	defer client.Close()

	serializer := NewAvroSerializer(client)
	groupID := createTestGroupID()
	artifactID := fmt.Sprintf("%s-autolookup", createTestArtifactID())

	testEvent := RegistrationTestEvent{
		EventID:   artifactID,
		Timestamp: time.Now().Unix(),
		Data:      "Register then auto lookup test",
		Version:   1,
	}

	// Parse schema for registration
	schemaContent := getRegistrationTestEventSchema()
	schema, err := avro.Parse(schemaContent)
	require.NoError(t, err)

	// Register the schema
	_, err = serializer.Client().RegisterSchema(ctx, groupID, artifactID, schema.String())
	require.NoError(t, err)

	// Use automatic lookup for serialization (no schema_id provided)
	serialized, err := serializer.Serialize(ctx, testEvent, schema)
	require.NoError(t, err)

	// Verify serialization worked
	assert.IsType(t, []byte{}, serialized)
	require.GreaterOrEqual(t, len(serialized), 5)
	assert.Equal(t, byte(0x0), serialized[0])

	// Test deserialization round trip
	var deserialized RegistrationTestEvent
	err = serializer.Deserialize(ctx, serialized, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, testEvent.EventID, deserialized.EventID)
	assert.Equal(t, testEvent.Timestamp, deserialized.Timestamp)
	assert.Equal(t, testEvent.Data, deserialized.Data)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
