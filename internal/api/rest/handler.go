package rest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gotrack/internal/model"
	"github.com/gotrack/internal/service"
)

type Handler struct {
	svc *service.EventService
}

func NewHandler(svc *service.EventService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) IngestEvents(c *gin.Context) {
	var req model.BatchEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:     "invalid request body: " + err.Error(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	authSource := c.GetString("source_id")
	for _, e := range req.Events {
		if e.SourceID != authSource {
			c.JSON(http.StatusForbidden, model.ErrorResponse{
				Error:     "api key not authorized for source_id: " + e.SourceID,
				RequestID: c.GetString("request_id"),
			})
			return
		}
	}

	resp := h.svc.IngestBatch(c.Request.Context(), req.Events)
	c.JSON(http.StatusAccepted, resp)
}

func (h *Handler) QueryEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	f := model.EventFilter{
		SourceID: c.Query("source"),
		Type:     c.Query("type"),
		Page:     page,
		PageSize: pageSize,
	}

	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			f.From = t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			f.To = t
		}
	}

	resp, err := h.svc.Query(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:     "query failed: " + err.Error(),
			RequestID: c.GetString("request_id"),
		})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) HealthCheck(c *gin.Context) {
	if err := h.svc.HealthCheck(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
