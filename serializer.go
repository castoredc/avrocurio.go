package avrocurio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hamba/avro/v2"
)

// AvroSerializer handles Avro serialization/deserialization using Confluent Schema Registry wire format.
type AvroSerializer struct {
	client *ApicurioClient
}

// NewAvroSerializer creates a new AvroSerializer with the given client.
func NewAvroSerializer(client *ApicurioClient) *AvroSerializer {
	return &AvroSerializer{client: client}
}

// SerializeWithSchemaID serializes an object using an explicit schema ID.
func (s *AvroSerializer) SerializeWithSchemaID(ctx context.Context, obj interface{}, schemaID uint32) ([]byte, error) {
	return s.serializeWithSchemaID(ctx, obj, schemaID)
}

// Serialize serializes an object with automatic schema lookup.
// Attempts to automatically find a matching schema in the registry.
func (s *AvroSerializer) Serialize(ctx context.Context, obj interface{}, schema avro.Schema) ([]byte, error) {
	// Automatic schema lookup
	schemaContent := schema.String()
	groupID, artifactID, err := s.client.FindArtifactByContent(ctx, schemaContent, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to find schema by content: %w", err)
	}

	if groupID == "" || artifactID == "" {
		return nil, fmt.Errorf("please register the schema manually or specify a schema_id: %w", ErrSchemaNotMatched)
	}

	// Find the exact schema ID that matches our content
	schemaID, err := s.client.FindExactSchemaVersionByContent(ctx, groupID, artifactID, schemaContent)
	if err != nil {
		return nil, fmt.Errorf("failed to find exact schema ID by content: %w", err)
	}

	return s.serializeWithSchemaID(ctx, obj, schemaID)
}

// serializeWithSchemaID performs core serialization logic using a specific schema ID.
func (s *AvroSerializer) serializeWithSchemaID(ctx context.Context, obj interface{}, schemaID uint32) ([]byte, error) {
	schemaMap, err := s.client.GetSchemaByGlobalID(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema by global ID: %w", err)
	}

	// Convert schema map to JSON string and parse as Avro schema
	schemaJSON, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	avroSchema, err := avro.Parse(string(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Avro schema: %w", err)
	}

	// Serialize the object
	avroPayload, err := avro.Marshal(avroSchema, obj)
	if err != nil {
		return nil, fmt.Errorf("data does not match schema: %w", err)
	}

	// Encode with Confluent wire format
	wireFormat := &ConfluentWireFormat{}
	return wireFormat.Encode(schemaID, avroPayload), nil
}

// Deserialize deserializes a message with Confluent wire format framing into target.
// Use DeserializeToInterface for raw deserialization.
func (s *AvroSerializer) Deserialize(ctx context.Context, message []byte, target interface{}) error {
	wireFormat := &ConfluentWireFormat{}
	schemaID, avroPayload, err := wireFormat.Decode(message)
	if err != nil {
		return fmt.Errorf("failed to decode wire format: %w", err)
	}

	schemaMap, err := s.client.GetSchemaByGlobalID(ctx, schemaID)
	if err != nil {
		return fmt.Errorf("failed to get schema by global ID: %w", err)
	}

	// Convert schema map to JSON string and parse as Avro schema
	schemaJSON, err := json.Marshal(schemaMap)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	writerSchema, err := avro.Parse(string(schemaJSON))
	if err != nil {
		return fmt.Errorf("failed to parse Avro writer schema: %w", err)
	}

	// If target is nil, deserialize to interface{}
	if target == nil {
		var result interface{}
		err = avro.Unmarshal(writerSchema, avroPayload, &result)
		if err != nil {
			return fmt.Errorf("failed to deserialize message: %w", err)
		}
		// Note: In Go, we can't return the result directly like in Python
		// The caller should use DeserializeToInterface instead
		return fmt.Errorf("use DeserializeToInterface for raw deserialization")
	}

	// Deserialize with reader schema (target object's schema)
	err = avro.Unmarshal(writerSchema, avroPayload, target)
	if err != nil {
		return fmt.Errorf("failed to deserialize message: %w", err)
	}

	return nil
}

// DeserializeToInterface deserializes a message to a raw interface{}.
func (s *AvroSerializer) DeserializeToInterface(ctx context.Context, message []byte) (interface{}, error) {
	wireFormat := &ConfluentWireFormat{}
	schemaID, avroPayload, err := wireFormat.Decode(message)
	if err != nil {
		return nil, fmt.Errorf("failed to decode wire format: %w", err)
	}

	schemaMap, err := s.client.GetSchemaByGlobalID(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema by global ID: %w", err)
	}

	// Convert schema map to JSON string and parse as Avro schema
	schemaJSON, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	writerSchema, err := avro.Parse(string(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Avro writer schema: %w", err)
	}

	var result interface{}
	err = avro.Unmarshal(writerSchema, avroPayload, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize message: %w", err)
	}

	return result, nil
}

// Client returns the underlying ApicurioClient for direct registry operations.
func (s *AvroSerializer) Client() *ApicurioClient {
	return s.client
}
