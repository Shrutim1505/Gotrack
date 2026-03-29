package grpc

import (
	"context"
	"fmt"
	"io"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gotrack/internal/model"
	"github.com/gotrack/internal/service"
	pb "github.com/gotrack/proto"
)

const maxStreamEvents = 10000

type Handler struct {
	pb.UnimplementedEventServiceServer
	svc *service.EventService
	log *zap.Logger
}

func NewHandler(svc *service.EventService, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) authenticate(ctx context.Context, apiKey string) (string, error) {
	if apiKey == "" {
		return "", status.Error(codes.Unauthenticated, "missing api key")
	}
	sourceID, err := h.svc.ValidateAPIKey(ctx, apiKey)
	if err != nil {
		return "", status.Error(codes.Unauthenticated, "invalid api key")
	}
	return sourceID, nil
}

func (h *Handler) IngestEvents(ctx context.Context, req *pb.IngestRequest) (*pb.IngestResponse, error) {
	sourceID, err := h.authenticate(ctx, req.ApiKey)
	if err != nil {
		return nil, err
	}

	inputs := make([]model.EventInput, 0, len(req.Events))
	for _, e := range req.Events {
		if e.SourceId != sourceID {
			return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("api key not authorized for source_id %q", e.SourceId))
		}
		inputs = append(inputs, model.EventInput{
			SourceID:  e.SourceId,
			Type:      e.Type,
			Payload:   e.Payload,
			Timestamp: e.Timestamp,
		})
	}

	resp := h.svc.IngestBatch(ctx, inputs)
	return &pb.IngestResponse{
		Accepted:    int32(resp.Accepted),
		Rejected:    int32(resp.Rejected),
		Duplicates:  int32(resp.Duplicates),
		RejectedIds: resp.RejectedIDs,
	}, nil
}

func (h *Handler) StreamEvents(stream pb.EventService_StreamEventsServer) error {
	var inputs []model.EventInput
	for {
		if len(inputs) >= maxStreamEvents {
			return status.Errorf(codes.ResourceExhausted, "stream exceeded max %d events", maxStreamEvents)
		}
		e, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		inputs = append(inputs, model.EventInput{
			SourceID:  e.SourceId,
			Type:      e.Type,
			Payload:   e.Payload,
			Timestamp: e.Timestamp,
		})
	}

	resp := h.svc.IngestBatch(stream.Context(), inputs)
	return stream.SendAndClose(&pb.IngestResponse{
		Accepted:    int32(resp.Accepted),
		Rejected:    int32(resp.Rejected),
		Duplicates:  int32(resp.Duplicates),
		RejectedIds: resp.RejectedIDs,
	})
}
