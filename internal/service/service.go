package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gotrack/internal/model"
	"github.com/gotrack/internal/validator"
	"github.com/gotrack/pkg/middleware"
)

type EventRepo interface {
	InsertEvent(ctx context.Context, e *model.Event) error
	InsertQuarantined(ctx context.Context, e *model.Event, reason string) error
	IsDuplicate(ctx context.Context, dedupKey string) (bool, error)
	QueryEvents(ctx context.Context, f model.EventFilter) ([]model.Event, int, error)
	ValidateAPIKey(ctx context.Context, key string) (string, error)
	Ping(ctx context.Context) error
}

type EventService struct {
	repo      EventRepo
	validator *validator.EventValidator
	log       *zap.Logger
}

func NewEventService(repo EventRepo, v *validator.EventValidator, log *zap.Logger) *EventService {
	return &EventService{repo: repo, validator: v, log: log}
}

func dedupKey(input model.EventInput) string {
	h := sha256.Sum256([]byte(input.SourceID + "|" + input.Type + "|" + input.Payload + "|" + input.Timestamp))
	return fmt.Sprintf("%x", h[:16])
}

func (s *EventService) IngestBatch(ctx context.Context, events []model.EventInput) model.IngestResponse {
	start := time.Now()
	resp := model.IngestResponse{}

	for _, input := range events {
		middleware.EventsReceived.Inc()

		eventID := uuid.New().String()
		ts := time.Now().UTC()
		if input.Timestamp != "" {
			if parsed, err := time.Parse(time.RFC3339, input.Timestamp); err == nil {
				ts = parsed
			}
		}

		event := &model.Event{
			ID:        eventID,
			SourceID:  input.SourceID,
			Type:      input.Type,
			Payload:   input.Payload,
			Timestamp: ts,
			Valid:     true,
			CreatedAt: time.Now().UTC(),
		}

		// Validate
		if err := s.validator.Validate(input); err != nil {
			middleware.EventsFailed.Inc()
			event.Valid = false
			_ = s.repo.InsertQuarantined(ctx, event, err.Error())
			resp.Rejected++
			resp.RejectedIDs = append(resp.RejectedIDs, eventID)
			s.log.Warn("event validation failed", zap.String("event_id", eventID), zap.Error(err))
			continue
		}

		// Dedup based on content hash
		dk := dedupKey(input)
		dup, err := s.repo.IsDuplicate(ctx, dk)
		if err != nil {
			s.log.Error("dedup check failed", zap.Error(err))
		}
		if dup {
			resp.Duplicates++
			continue
		}

		// Store
		if err := s.repo.InsertEvent(ctx, event); err != nil {
			middleware.EventsFailed.Inc()
			resp.Rejected++
			s.log.Error("failed to store event", zap.Error(err))
			continue
		}
		resp.Accepted++
	}

	middleware.IngestionLatency.Observe(time.Since(start).Seconds())
	return resp
}

func (s *EventService) Query(ctx context.Context, f model.EventFilter) (*model.PaginatedResponse, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 50
	}

	events, total, err := s.repo.QueryEvents(ctx, f)
	if err != nil {
		return nil, err
	}

	return &model.PaginatedResponse{
		Data:       events,
		Page:       f.Page,
		PageSize:   f.PageSize,
		TotalCount: total,
	}, nil
}

func (s *EventService) ValidateAPIKey(ctx context.Context, key string) (string, error) {
	return s.repo.ValidateAPIKey(ctx, key)
}

func (s *EventService) HealthCheck(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
