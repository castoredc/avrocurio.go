package avrocurio

import (
	"bytes"
	"testing"
)

func TestConfluentWireFormat_Encode(t *testing.T) {
	cwf := &ConfluentWireFormat{}

	tests := []struct {
		name     string
		schemaID uint32
		payload  []byte
		expected []byte
	}{
		{
			name:     "basic encoding",
			schemaID: 12345,
			payload:  []byte("test payload"),
			expected: append([]byte{0x0, 0x00, 0x00, 0x30, 0x39}, []byte("test payload")...),
		},
		{
			name:     "empty payload",
			schemaID: 999,
			payload:  []byte(""),
			expected: []byte{0x0, 0x00, 0x00, 0x03, 0xe7},
		},
		{
			name:     "large schema ID",
			schemaID: 2147483647, // Max uint32/2
			payload:  []byte("test"),
			expected: append([]byte{0x0, 0x7f, 0xff, 0xff, 0xff}, []byte("test")...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cwf.Encode(tt.schemaID, tt.payload)

			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Encode() = %v, want %v", result, tt.expected)
			}

			// Verify structure
			if result[0] != MagicByte {
				t.Errorf("Magic byte = %v, want %v", result[0], MagicByte)
			}

			if len(result) != HeaderSize+len(tt.payload) {
				t.Errorf("Result length = %d, want %d", len(result), HeaderSize+len(tt.payload))
			}

			// Verify payload is at the end
			if !bytes.Equal(result[HeaderSize:], tt.payload) {
				t.Errorf("Payload = %v, want %v", result[HeaderSize:], tt.payload)
			}
		})
	}
}

func TestConfluentWireFormat_Decode(t *testing.T) {
	cwf := &ConfluentWireFormat{}

	tests := []struct {
		name            string
		message         []byte
		expectedID      uint32
		expectedPayload []byte
		expectedError   bool
	}{
		{
			name:            "basic decoding",
			message:         append([]byte{0x0, 0x00, 0x00, 0x30, 0x39}, []byte("test payload")...),
			expectedID:      12345,
			expectedPayload: []byte("test payload"),
			expectedError:   false,
		},
		{
			name:            "empty payload",
			message:         []byte{0x0, 0x00, 0x00, 0x03, 0xe7},
			expectedID:      999,
			expectedPayload: []byte(""),
			expectedError:   false,
		},
		{
			name:            "invalid magic byte",
			message:         []byte{0x1, 0x00, 0x00, 0x30, 0x39, 0x74, 0x65, 0x73, 0x74},
			expectedID:      0,
			expectedPayload: nil,
			expectedError:   true,
		},
		{
			name:            "message too short",
			message:         []byte{0x0, 0x00, 0x00},
			expectedID:      0,
			expectedPayload: nil,
			expectedError:   true,
		},
		{
			name:            "empty message",
			message:         []byte{},
			expectedID:      0,
			expectedPayload: nil,
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaID, payload, err := cwf.Decode(tt.message)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Decode() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Decode() unexpected error: %v", err)
				return
			}

			if schemaID != tt.expectedID {
				t.Errorf("Schema ID = %v, want %v", schemaID, tt.expectedID)
			}

			if !bytes.Equal(payload, tt.expectedPayload) {
				t.Errorf("Payload = %v, want %v", payload, tt.expectedPayload)
			}
		})
	}
}

func TestConfluentWireFormat_ValidateMagicByte(t *testing.T) {
	cwf := &ConfluentWireFormat{}

	tests := []struct {
		name          string
		message       []byte
		expectedError bool
	}{
		{
			name:          "valid magic byte",
			message:       []byte{0x0, 0x00, 0x00, 0x30, 0x39, 0x74, 0x65, 0x73, 0x74},
			expectedError: false,
		},
		{
			name:          "invalid magic byte",
			message:       []byte{0x1, 0x00, 0x00, 0x30, 0x39, 0x74, 0x65, 0x73, 0x74},
			expectedError: true,
		},
		{
			name:          "empty message",
			message:       []byte{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cwf.ValidateMagicByte(tt.message)

			if tt.expectedError && err == nil {
				t.Errorf("ValidateMagicByte() expected error but got none")
			}

			if !tt.expectedError && err != nil {
				t.Errorf("ValidateMagicByte() unexpected error: %v", err)
			}
		})
	}
}

func TestWireFormatRoundTrip(t *testing.T) {
	testCases := []struct {
		schemaID uint32
		payload  []byte
	}{
		{1, []byte("simple")},
		{999999, []byte("complex payload with \x00 null bytes")},
		{0, []byte("")},
		{2147483647, []byte("max schema id test")},
	}

	cwf := &ConfluentWireFormat{}

	for _, tc := range testCases {
		t.Run("round trip", func(t *testing.T) {
			encoded := cwf.Encode(tc.schemaID, tc.payload)
			decodedSchemaID, decodedPayload, err := cwf.Decode(encoded)
			if err != nil {
				t.Errorf("Round trip failed: %v", err)
			}

			if decodedSchemaID != tc.schemaID {
				t.Errorf("Schema ID = %v, want %v", decodedSchemaID, tc.schemaID)
			}

			if !bytes.Equal(decodedPayload, tc.payload) {
				t.Errorf("Payload = %v, want %v", decodedPayload, tc.payload)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if MagicByte != 0x0 {
		t.Errorf("MagicByte = %v, want 0x0", MagicByte)
	}

	if SchemaIDSize != 4 {
		t.Errorf("SchemaIDSize = %v, want 4", SchemaIDSize)
	}

	if HeaderSize != 5 {
		t.Errorf("HeaderSize = %v, want 5", HeaderSize)
	}
}

func TestPackageFunctions(t *testing.T) {
	schemaID := uint32(12345)
	payload := []byte("test payload")

	// Test package-level Encode function
	encoded := Encode(schemaID, payload)
	if encoded[0] != MagicByte {
		t.Errorf("Package Encode() magic byte = %v, want %v", encoded[0], MagicByte)
	}

	// Test package-level Decode function
	decodedID, decodedPayload, err := Decode(encoded)
	if err != nil {
		t.Errorf("Package Decode() error: %v", err)
	}

	if decodedID != schemaID {
		t.Errorf("Package Decode() schema ID = %v, want %v", decodedID, schemaID)
	}

	if !bytes.Equal(decodedPayload, payload) {
		t.Errorf("Package Decode() payload = %v, want %v", decodedPayload, payload)
	}

	// Test package-level ValidateMagicByte function
	if err := ValidateMagicByte(encoded); err != nil {
		t.Errorf("Package ValidateMagicByte() error: %v", err)
	}

	// Test with invalid magic byte
	invalid := []byte{0x1, 0x00, 0x00, 0x30, 0x39}
	if err := ValidateMagicByte(invalid); err == nil {
		t.Errorf("Package ValidateMagicByte() expected error for invalid magic byte")
	}
}
