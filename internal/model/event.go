package model

import "time"

type Event struct {
	ID        string    `json:"id" db:"id"`
	SourceID  string    `json:"source_id" db:"source_id"`
	Type      string    `json:"type" db:"type"`
	Payload   string    `json:"payload" db:"payload"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	Valid     bool      `json:"valid" db:"valid"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type BatchEventRequest struct {
	Events []EventInput `json:"events" binding:"required,min=1,max=1000"`
}

type EventInput struct {
	SourceID  string `json:"source_id" binding:"required"`
	Type      string `json:"type" binding:"required"`
	Payload   string `json:"payload" binding:"required,max=65536"`
	Timestamp string `json:"timestamp"`
}

type EventFilter struct {
	SourceID string
	Type     string
	From     time.Time
	To       time.Time
	Page     int
	PageSize int
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalCount int         `json:"total_count"`
}

type ErrorResponse struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

type IngestResponse struct {
	Accepted    int      `json:"accepted"`
	Rejected    int      `json:"rejected"`
	Duplicates  int      `json:"duplicates"`
	RejectedIDs []string `json:"rejected_ids"`
}

type APIKey struct {
	Key      string `json:"key" db:"key"`
	SourceID string `json:"source_id" db:"source_id"`
	Active   bool   `json:"active" db:"active"`
}
