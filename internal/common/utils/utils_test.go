package utils

import (
	"testing"
)

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid bucket names
		{"valid simple name", "my-bucket", true},
		{"valid with numbers", "bucket123", true},
		{"valid min length", "abc", true},
		{"valid with hyphen", "my-test-bucket", true},
		{"valid all numbers", "123456789", true}, // Numbers are alphanumeric, so this is valid
		
		// Invalid bucket names
		{"too short", "ab", false},
		{"too long", "a123456789012345678901234567890123456789012345678901234567890123", false},
		{"starts with hyphen", "-bucket", false},
		{"ends with hyphen", "bucket-", false},
		{"uppercase letters", "MyBucket", false},
		{"contains underscore", "my_bucket", false},
		{"contains space", "my bucket", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateBucketName(tt.input)
			if result != tt.expected {
				t.Errorf("ValidateBucketName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateObjectKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid object keys
		{"valid simple key", "file.txt", true},
		{"valid with path", "folder/file.txt", true},
		{"valid with special chars", "my-file_v1.2.txt", true},
		{"valid single char", "a", true},
		{"valid with spaces", "my file.txt", true},
		
		// Invalid object keys
		{"empty string", "", false},
		{"only whitespace", "   ", false},
		{"only tabs", "\t\t", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateObjectKey(tt.input)
			if result != tt.expected {
				t.Errorf("ValidateObjectKey(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateObjectKey(t *testing.T) {
	key1 := GenerateObjectKey()
	key2 := GenerateObjectKey()

	// Check prefix
	if len(key1) < 4 || key1[:4] != "obj_" {
		t.Errorf("GenerateObjectKey() = %q, should start with 'obj_'", key1)
	}

	// Check uniqueness
	if key1 == key2 {
		t.Errorf("GenerateObjectKey() generated duplicate keys: %q", key1)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()

	// Check not empty
	if id1 == "" {
		t.Error("GenerateID() returned empty string")
	}

	// Check uniqueness
	if id1 == id2 {
		t.Errorf("GenerateID() generated duplicate IDs: %q", id1)
	}

	// Check UUID format (36 chars with hyphens)
	if len(id1) != 36 {
		t.Errorf("GenerateID() = %q, expected UUID format with 36 characters", id1)
	}
}
