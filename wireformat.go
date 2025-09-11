package avrocurio

import (
	"encoding/binary"
	"fmt"
)

const (
	// MagicByte is the magic byte that identifies the Confluent wire format.
	MagicByte = 0x0
	// SchemaIDSize is the size of the schema ID in bytes (4 bytes, big-endian).
	SchemaIDSize = 4
	// HeaderSize is the total size of the wire format header (magic byte + schema ID).
	HeaderSize = 1 + SchemaIDSize
)

// ConfluentWireFormat handles Confluent Schema Registry wire format encoding/decoding.
//
// The wire format consists of:
// - Magic byte (0x0) - 1 byte
// - Schema ID - 4 bytes (big-endian)
// - Avro payload - remaining bytes
type ConfluentWireFormat struct{}

// Encode encodes payload with Confluent wire format.
func (cwf *ConfluentWireFormat) Encode(schemaID uint32, payload []byte) []byte {
	header := make([]byte, HeaderSize)
	header[0] = MagicByte
	binary.BigEndian.PutUint32(header[1:], schemaID)

	message := make([]byte, len(header)+len(payload))
	copy(message, header)
	copy(message[HeaderSize:], payload)

	return message
}

// Decode decodes message with Confluent wire format.
func (cwf *ConfluentWireFormat) Decode(message []byte) (uint32, []byte, error) {
	if err := cwf.ValidateMagicByte(message); err != nil {
		return 0, nil, err
	}

	if len(message) < HeaderSize {
		return 0, nil, fmt.Errorf("message too short, expected at least %d bytes, got %d: %w",
			HeaderSize, len(message), ErrInvalidWireFormat)
	}

	schemaID := binary.BigEndian.Uint32(message[1:HeaderSize])
	payload := message[HeaderSize:]

	return schemaID, payload, nil
}

// ValidateMagicByte validates that message starts with correct magic byte.
func (cwf *ConfluentWireFormat) ValidateMagicByte(message []byte) error {
	if len(message) < 1 {
		return fmt.Errorf("message is empty: %w", ErrInvalidWireFormat)
	}

	magicByte := message[0]
	if magicByte != MagicByte {
		return fmt.Errorf("invalid magic byte 0x%02X, expected 0x%02X: %w",
			magicByte, MagicByte, ErrInvalidWireFormat)
	}

	return nil
}

// Package-level convenience functions

// Encode is a convenience function for encoding with Confluent wire format.
func Encode(schemaID uint32, payload []byte) []byte {
	cwf := &ConfluentWireFormat{}
	return cwf.Encode(schemaID, payload)
}

// Decode is a convenience function for decoding with Confluent wire format.
func Decode(message []byte) (uint32, []byte, error) {
	cwf := &ConfluentWireFormat{}
	return cwf.Decode(message)
}

// ValidateMagicByte is a convenience function for validating magic byte.
func ValidateMagicByte(message []byte) error {
	cwf := &ConfluentWireFormat{}
	return cwf.ValidateMagicByte(message)
}
