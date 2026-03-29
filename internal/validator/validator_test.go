package validator

import (
	"strings"
	"testing"

	"github.com/gotrack/internal/model"
)

func TestValidate_Success(t *testing.T) {
	v := New()
	err := v.Validate(model.EventInput{
		SourceID: "src-1", Type: "click", Payload: `{"page":"home"}`,
	})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_MissingSourceID(t *testing.T) {
	v := New()
	err := v.Validate(model.EventInput{Type: "click", Payload: `{}`})
	if err == nil || !strings.Contains(err.Error(), "source_id") {
		t.Errorf("expected source_id error, got %v", err)
	}
}

func TestValidate_MissingType(t *testing.T) {
	v := New()
	err := v.Validate(model.EventInput{SourceID: "s1", Payload: `{}`})
	if err == nil || !strings.Contains(err.Error(), "type") {
		t.Errorf("expected type error, got %v", err)
	}
}

func TestValidate_EmptyPayload(t *testing.T) {
	v := New()
	err := v.Validate(model.EventInput{SourceID: "s1", Type: "click", Payload: ""})
	if err == nil {
		t.Error("expected error for empty payload")
	}
}

func TestValidate_InvalidJSON(t *testing.T) {
	v := New()
	err := v.Validate(model.EventInput{SourceID: "s1", Type: "click", Payload: "not-json"})
	if err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Errorf("expected JSON error, got %v", err)
	}
}

func TestValidate_PayloadTooLarge(t *testing.T) {
	v := New()
	big := `{"data":"` + strings.Repeat("x", MaxPayloadSize) + `"}`
	err := v.Validate(model.EventInput{SourceID: "s1", Type: "click", Payload: big})
	if err == nil || !strings.Contains(err.Error(), "max size") {
		t.Errorf("expected max size error, got %v", err)
	}
}
