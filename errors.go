package avrocurio

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrSchemaNotFound    = errors.New("schema not found")
	ErrInvalidWireFormat = errors.New("invalid wire format")
	ErrSchemaNotMatched  = errors.New("no matching schema found for automatic lookup")
)

// SchemaRegistrationError contains context about failed schema registration
type SchemaRegistrationError struct {
	Group        string
	ArtifactName string
	Err          error
}

func (e *SchemaRegistrationError) Error() string {
	return fmt.Sprintf("failed to register schema %s/%s: %v", e.Group, e.ArtifactName, e.Err)
}

func (e *SchemaRegistrationError) Unwrap() error {
	return e.Err
}
