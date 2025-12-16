package enums

import (
	"fmt"
)

// EventType represents the type of webhook event
type EventType string

const (
	// EventTypeCredit represents a credit/postback event
	EventTypeCredit EventType = "CREDIT"

	// EventTypeDebit represents a debit/chargeback event
	EventTypeDebit EventType = "DEBIT"
)

// IsValid checks if the event type is valid
func (e EventType) IsValid() bool {
	switch e {
	case EventTypeCredit, EventTypeDebit:
		return true
	default:
		return false
	}
}

// Validate validates the event type and returns an error if invalid
func (e EventType) Validate() error {
	if !e.IsValid() {
		return fmt.Errorf("invalid event type: %s (must be one of: %s, %s)",
			e, EventTypeCredit, EventTypeDebit)
	}
	return nil
}
