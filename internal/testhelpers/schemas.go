package testhelpers

import "github.com/hamba/avro/v2"

// TestUser represents a simple test user for integration tests
type TestUser struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

// SimpleUserSchema returns a simple Avro schema for testing
func SimpleUserSchema() (avro.Schema, error) {
	schemaStr := `{
		"type": "record",
		"name": "SimpleUser",
		"fields": [
			{"name": "name", "type": "string"},
			{"name": "age", "type": "int"}
		]
	}`
	return avro.Parse(schemaStr)
}

// ComplexUserSchema returns a complex Avro schema for testing
func ComplexUserSchema() (avro.Schema, error) {
	schemaStr := `{
		"type": "record",
		"name": "ComplexUser",
		"fields": [
			{"name": "name", "type": "string"},
			{"name": "age", "type": "int"},
			{"name": "email", "type": ["null", "string"], "default": null},
			{"name": "is_active", "type": "boolean", "default": true}
		]
	}`
	return avro.Parse(schemaStr)
}

// ProductSchema returns a product schema for testing
func ProductSchema() (avro.Schema, error) {
	schemaStr := `{
		"type": "record",
		"name": "Product",
		"fields": [
			{"name": "id", "type": "int"},
			{"name": "name", "type": "string"},
			{"name": "price", "type": "float"},
			{"name": "category", "type": "string"}
		]
	}`
	return avro.Parse(schemaStr)
}

// TestSchemas contains all test schemas for convenience
var TestSchemas = map[string]func() (avro.Schema, error){
	"simple_user":  SimpleUserSchema,
	"complex_user": ComplexUserSchema,
	"product":      ProductSchema,
}
