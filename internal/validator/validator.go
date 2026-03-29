package validator

import (
	"encoding/json"
	"fmt"
	"github.com/gotrack/internal/model"
)

const MaxPayloadSize = 65536

type EventValidator struct{}

func New() *EventValidator { return &EventValidator{} }

func (v *EventValidator) Validate(e model.EventInput) error {
	if e.SourceID == "" {
		return fmt.Errorf("source_id is required")
	}
	if e.Type == "" {
		return fmt.Errorf("type is required")
	}
	if len(e.Payload) == 0 {
		return fmt.Errorf("payload is required")
	}
	if len(e.Payload) > MaxPayloadSize {
		return fmt.Errorf("payload exceeds max size of %d bytes", MaxPayloadSize)
	}
	if !json.Valid([]byte(e.Payload)) {
		return fmt.Errorf("payload must be valid JSON")
	}
	return nil
}
