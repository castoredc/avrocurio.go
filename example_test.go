//go:build integration

package avrocurio_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/castoredc/avrocurio.go"
	"github.com/hamba/avro/v2"
)

// User represents our data model
type User struct {
	Name  string `avro:"name"`
	Age   int    `avro:"age"`
	Email string `avro:"email"`
}

// ExampleAvroSerializer demonstrates basic usage of the Avro serializer
// with Apicurio Registry for schema registration, serialization, and deserialization.
func ExampleAvroSerializer() {
	// Configure connection to Apicurio Registry
	config := avrocurio.NewApicurioConfig().
		WithBaseURL("http://localhost:8080").
		WithTimeout(30 * time.Second).              // Time-out API requests after 30s
		WithMaxRetries(5).                          // Retry up to 5 times
		WithRetryBaseDelay(200 * time.Millisecond). // Start with 200ms delay
		WithRetryMaxDelay(30 * time.Second)         // Cap delays at 30s

	// Create client and serializer
	client, err := avrocurio.NewApicurioClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	serializer := avrocurio.NewAvroSerializer(client)

	// Define Avro schema
	schemaStr := `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "name", "type": "string"},
			{"name": "age", "type": "int"},
			{"name": "email", "type": "string"}
		]
	}`

	schema, err := avro.Parse(schemaStr)
	if err != nil {
		log.Fatalf("Failed to parse schema: %v", err)
	}

	ctx := context.Background()

	// Register the schema
	globalID, err := client.RegisterSchema(ctx, "default", "user-schema", schema.String())
	if err != nil {
		log.Fatalf("Failed to register schema: %v", err)
	}
	fmt.Printf("Registered schema with global ID: %d\n", globalID)

	// Create a user instance
	user := User{
		Name:  "John Doe",
		Age:   30,
		Email: "john@example.com",
	}

	// Serialize the user to an Avro binary with Confluent registry framing
	// using the registered schema ID
	serialized, err := serializer.SerializeWithSchemaID(ctx, user, globalID)
	if err != nil {
		log.Fatalf("Failed to serialize user: %v", err)
	}
	fmt.Printf("Serialized user: %d bytes\n", len(serialized))

	// Typically however, you would serialize with automatic schema lookup
	// instead.
	serialized2, err := serializer.Serialize(ctx, user, schema)
	if err != nil {
		log.Fatalf("Failed to serialize with automatic lookup: %v", err)
	}
	fmt.Printf("Serialized with automatic lookup: %d bytes\n", len(serialized2))

	// Deserialize the binary back to a User instance
	var deserializedUser User
	err = serializer.Deserialize(ctx, serialized, &deserializedUser)
	if err != nil {
		log.Fatalf("Failed to deserialize user: %v", err)
	}
	fmt.Printf("Deserialized user: %+v\n", deserializedUser)

	// Alternatively, deserialize to a raw interface{}
	rawData, err := serializer.DeserializeToInterface(ctx, serialized)
	if err != nil {
		log.Fatalf("Failed to deserialize to interface{}: %v", err)
	}
	fmt.Printf("Raw deserialized data: %+v\n", rawData)
}
