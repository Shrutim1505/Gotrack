package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/gotrack/internal/model"
)

type EventRepository struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewEventRepository(db *pgxpool.Pool, redis *redis.Client) *EventRepository {
	return &EventRepository{db: db, redis: redis}
}

func (r *EventRepository) InsertEvent(ctx context.Context, e *model.Event) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO events (id, source_id, type, payload, timestamp, valid, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.ID, e.SourceID, e.Type, e.Payload, e.Timestamp, e.Valid, e.CreatedAt)
	return err
}

func (r *EventRepository) InsertQuarantined(ctx context.Context, e *model.Event, reason string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO quarantined_events (id, source_id, type, payload, timestamp, reason, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.ID, e.SourceID, e.Type, e.Payload, e.Timestamp, reason, e.CreatedAt)
	return err
}

func (r *EventRepository) IsDuplicate(ctx context.Context, eventID string) (bool, error) {
	key := "dedup:" + eventID
	res, err := r.redis.SetNX(ctx, key, 1, 24*time.Hour).Result()
	if err != nil {
		return false, err
	}
	return !res, nil // SetNX returns false if key already existed
}

func (r *EventRepository) QueryEvents(ctx context.Context, f model.EventFilter) ([]model.Event, int, error) {
	where := "WHERE valid = true"
	args := []interface{}{}
	idx := 1

	if f.SourceID != "" {
		where += fmt.Sprintf(" AND source_id = $%d", idx)
		args = append(args, f.SourceID)
		idx++
	}
	if f.Type != "" {
		where += fmt.Sprintf(" AND type = $%d", idx)
		args = append(args, f.Type)
		idx++
	}
	if !f.From.IsZero() {
		where += fmt.Sprintf(" AND timestamp >= $%d", idx)
		args = append(args, f.From)
		idx++
	}
	if !f.To.IsZero() {
		where += fmt.Sprintf(" AND timestamp <= $%d", idx)
		args = append(args, f.To)
		idx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM events " + where
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf("SELECT id, source_id, type, payload, timestamp, valid, created_at FROM events %s ORDER BY timestamp DESC LIMIT $%d OFFSET $%d", where, idx, idx+1)
	args = append(args, f.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var e model.Event
		if err := rows.Scan(&e.ID, &e.SourceID, &e.Type, &e.Payload, &e.Timestamp, &e.Valid, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, nil
}

func (r *EventRepository) ValidateAPIKey(ctx context.Context, key string) (string, error) {
	// Check Redis cache first
	cached, err := r.redis.Get(ctx, "apikey:"+key).Result()
	if err == nil {
		return cached, nil
	}

	var sourceID string
	err = r.db.QueryRow(ctx, "SELECT source_id FROM api_keys WHERE key = $1 AND active = true", key).Scan(&sourceID)
	if err != nil {
		return "", fmt.Errorf("invalid api key")
	}

	// Cache for 5 minutes
	r.redis.Set(ctx, "apikey:"+key, sourceID, 5*time.Minute)
	return sourceID, nil
}

func (r *EventRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}
