package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/gotrack/internal/model"
	"github.com/gotrack/internal/validator"
)

// --- Mock Repository ---

type mockRepo struct {
	events       map[string]*model.Event
	quarantined  map[string]*model.Event
	dedupSet     map[string]bool
	failInsert   bool
	failQuery    bool
	failPing     bool
	apiKeys      map[string]string
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		events:      make(map[string]*model.Event),
		quarantined: make(map[string]*model.Event),
		dedupSet:    make(map[string]bool),
		apiKeys:     map[string]string{"valid-key": "source-1"},
	}
}

func (m *mockRepo) InsertEvent(_ context.Context, e *model.Event) error {
	if m.failInsert {
		return errors.New("insert failed")
	}
	m.events[e.ID] = e
	return nil
}

func (m *mockRepo) InsertQuarantined(_ context.Context, e *model.Event, _ string) error {
	m.quarantined[e.ID] = e
	return nil
}

func (m *mockRepo) IsDuplicate(_ context.Context, dedupKey string) (bool, error) {
	if m.dedupSet[dedupKey] {
		return true, nil
	}
	m.dedupSet[dedupKey] = true
	return false, nil
}

func (m *mockRepo) QueryEvents(_ context.Context, f model.EventFilter) ([]model.Event, int, error) {
	if m.failQuery {
		return nil, 0, errors.New("query failed")
	}
	var result []model.Event
	for _, e := range m.events {
		if f.SourceID != "" && e.SourceID != f.SourceID {
			continue
		}
		if f.Type != "" && e.Type != f.Type {
			continue
		}
		result = append(result, *e)
	}
	return result, len(result), nil
}

func (m *mockRepo) ValidateAPIKey(_ context.Context, key string) (string, error) {
	if src, ok := m.apiKeys[key]; ok {
		return src, nil
	}
	return "", errors.New("invalid api key")
}

func (m *mockRepo) Ping(_ context.Context) error {
	if m.failPing {
		return errors.New("db down")
	}
	return nil
}

// --- Tests ---

func newTestService(repo *mockRepo) *EventService {
	return NewEventService(repo, validator.New(), zap.NewNop())
}

func validInput() model.EventInput {
	return model.EventInput{
		SourceID:  "src-1",
		Type:      "click",
		Payload:   `{"page":"home"}`,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func TestIngestBatch_Success(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	resp := svc.IngestBatch(context.Background(), []model.EventInput{validInput(), validInput()})

	if resp.Accepted != 2 {
		t.Errorf("expected 2 accepted, got %d", resp.Accepted)
	}
	if resp.Rejected != 0 {
		t.Errorf("expected 0 rejected, got %d", resp.Rejected)
	}
	if len(repo.events) != 2 {
		t.Errorf("expected 2 stored events, got %d", len(repo.events))
	}
}

func TestIngestBatch_ValidationFailure(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	bad := model.EventInput{SourceID: "", Type: "click", Payload: `{}`}
	resp := svc.IngestBatch(context.Background(), []model.EventInput{bad})

	if resp.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", resp.Rejected)
	}
	if resp.Accepted != 0 {
		t.Errorf("expected 0 accepted, got %d", resp.Accepted)
	}
	if len(repo.quarantined) != 1 {
		t.Errorf("expected 1 quarantined event, got %d", len(repo.quarantined))
	}
}

func TestIngestBatch_InvalidPayload(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	bad := model.EventInput{SourceID: "s1", Type: "click", Payload: "not-json"}
	resp := svc.IngestBatch(context.Background(), []model.EventInput{bad})

	if resp.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", resp.Rejected)
	}
}

func TestIngestBatch_InsertFailure(t *testing.T) {
	repo := newMockRepo()
	repo.failInsert = true
	svc := newTestService(repo)

	resp := svc.IngestBatch(context.Background(), []model.EventInput{validInput()})

	if resp.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", resp.Rejected)
	}
	if resp.Accepted != 0 {
		t.Errorf("expected 0 accepted, got %d", resp.Accepted)
	}
}

func TestIngestBatch_EmptyPayload(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	bad := model.EventInput{SourceID: "s1", Type: "click", Payload: ""}
	resp := svc.IngestBatch(context.Background(), []model.EventInput{bad})

	if resp.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", resp.Rejected)
	}
}

func TestIngestBatch_MixedValid(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	good := validInput()
	bad := model.EventInput{SourceID: "", Type: "", Payload: ""}
	resp := svc.IngestBatch(context.Background(), []model.EventInput{good, bad})

	if resp.Accepted != 1 {
		t.Errorf("expected 1 accepted, got %d", resp.Accepted)
	}
	if resp.Rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", resp.Rejected)
	}
}

func TestIngestBatch_NoTimestamp(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	input := model.EventInput{SourceID: "s1", Type: "click", Payload: `{"a":1}`}
	resp := svc.IngestBatch(context.Background(), []model.EventInput{input})

	if resp.Accepted != 1 {
		t.Errorf("expected 1 accepted, got %d", resp.Accepted)
	}
}

func TestQuery_Defaults(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	// Insert an event first
	svc.IngestBatch(context.Background(), []model.EventInput{validInput()})

	resp, err := svc.Query(context.Background(), model.EventFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
	if resp.PageSize != 50 {
		t.Errorf("expected page_size 50, got %d", resp.PageSize)
	}
}

func TestQuery_WithFilter(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	svc.IngestBatch(context.Background(), []model.EventInput{validInput()})

	resp, err := svc.Query(context.Background(), model.EventFilter{SourceID: "src-1", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 1 {
		t.Errorf("expected 1 result, got %d", resp.TotalCount)
	}
}

func TestQuery_Error(t *testing.T) {
	repo := newMockRepo()
	repo.failQuery = true
	svc := newTestService(repo)

	_, err := svc.Query(context.Background(), model.EventFilter{})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestQuery_PageSizeBounds(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	resp, _ := svc.Query(context.Background(), model.EventFilter{PageSize: 999})
	if resp.PageSize != 50 {
		t.Errorf("expected clamped page_size 50, got %d", resp.PageSize)
	}

	resp, _ = svc.Query(context.Background(), model.EventFilter{PageSize: -1})
	if resp.PageSize != 50 {
		t.Errorf("expected default page_size 50, got %d", resp.PageSize)
	}
}

func TestHealthCheck_OK(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	if err := svc.HealthCheck(context.Background()); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestHealthCheck_Fail(t *testing.T) {
	repo := newMockRepo()
	repo.failPing = true
	svc := newTestService(repo)

	if err := svc.HealthCheck(context.Background()); err == nil {
		t.Error("expected error, got nil")
	}
}
