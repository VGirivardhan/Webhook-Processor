package enums

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventType_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		expected  bool
	}{
		{
			name:      "valid credit event type",
			eventType: EventTypeCredit,
			expected:  true,
		},
		{
			name:      "valid debit event type",
			eventType: EventTypeDebit,
			expected:  true,
		},
		{
			name:      "invalid empty event type",
			eventType: EventType(""),
			expected:  false,
		},
		{
			name:      "invalid unknown event type",
			eventType: EventType("UNKNOWN"),
			expected:  false,
		},
		{
			name:      "invalid lowercase credit",
			eventType: EventType("credit"),
			expected:  false,
		},
		{
			name:      "invalid lowercase debit",
			eventType: EventType("debit"),
			expected:  false,
		},
		{
			name:      "invalid mixed case",
			eventType: EventType("Credit"),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.eventType.IsValid()
			assert.Equal(t, tt.expected, result, "EventType.IsValid() should return %v for %s", tt.expected, tt.eventType)
		})
	}
}

func TestEventType_Validate(t *testing.T) {
	tests := []struct {
		name        string
		eventType   EventType
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid credit event type",
			eventType:   EventTypeCredit,
			expectError: false,
		},
		{
			name:        "valid debit event type",
			eventType:   EventTypeDebit,
			expectError: false,
		},
		{
			name:        "invalid empty event type",
			eventType:   EventType(""),
			expectError: true,
			errorMsg:    "invalid event type:  (must be one of: CREDIT, DEBIT)",
		},
		{
			name:        "invalid unknown event type",
			eventType:   EventType("UNKNOWN"),
			expectError: true,
			errorMsg:    "invalid event type: UNKNOWN (must be one of: CREDIT, DEBIT)",
		},
		{
			name:        "invalid lowercase credit",
			eventType:   EventType("credit"),
			expectError: true,
			errorMsg:    "invalid event type: credit (must be one of: CREDIT, DEBIT)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.eventType.Validate()

			if tt.expectError {
				assert.Error(t, err, "EventType.Validate() should return an error for %s", tt.eventType)
				if tt.errorMsg != "" {
					assert.Equal(t, tt.errorMsg, err.Error(), "Error message should match expected")
				}
			} else {
				assert.NoError(t, err, "EventType.Validate() should not return an error for %s", tt.eventType)
			}
		})
	}
}

func TestEventType_Constants(t *testing.T) {
	t.Run("event type constants should have correct values", func(t *testing.T) {
		assert.Equal(t, EventType("CREDIT"), EventTypeCredit, "EventTypeCredit should equal 'CREDIT'")
		assert.Equal(t, EventType("DEBIT"), EventTypeDebit, "EventTypeDebit should equal 'DEBIT'")
	})

	t.Run("event type constants should be valid", func(t *testing.T) {
		assert.True(t, EventTypeCredit.IsValid(), "EventTypeCredit should be valid")
		assert.True(t, EventTypeDebit.IsValid(), "EventTypeDebit should be valid")

		assert.NoError(t, EventTypeCredit.Validate(), "EventTypeCredit should validate without error")
		assert.NoError(t, EventTypeDebit.Validate(), "EventTypeDebit should validate without error")
	})
}

func TestEventType_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		valid     bool
	}{
		{
			name:      "whitespace only",
			eventType: EventType("   "),
			valid:     false,
		},
		{
			name:      "credit with whitespace",
			eventType: EventType(" CREDIT "),
			valid:     false,
		},
		{
			name:      "debit with whitespace",
			eventType: EventType(" DEBIT "),
			valid:     false,
		},
		{
			name:      "special characters",
			eventType: EventType("CREDIT!"),
			valid:     false,
		},
		{
			name:      "numeric",
			eventType: EventType("123"),
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.eventType.IsValid(), "EventType.IsValid() result should match expected for %s", tt.eventType)

			if tt.valid {
				assert.NoError(t, tt.eventType.Validate(), "Valid event type should not return validation error")
			} else {
				assert.Error(t, tt.eventType.Validate(), "Invalid event type should return validation error")
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkEventType_IsValid(b *testing.B) {
	eventType := EventTypeCredit

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = eventType.IsValid()
	}
}

func BenchmarkEventType_Validate(b *testing.B) {
	eventType := EventTypeCredit

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = eventType.Validate()
	}
}

func BenchmarkEventType_IsValid_Invalid(b *testing.B) {
	eventType := EventType("INVALID")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = eventType.IsValid()
	}
}

func BenchmarkEventType_Validate_Invalid(b *testing.B) {
	eventType := EventType("INVALID")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = eventType.Validate()
	}
}
