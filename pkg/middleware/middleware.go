package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	EventsReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_received_total",
		Help: "Total events received",
	})
	EventsFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "events_failed_total",
		Help: "Total events that failed validation",
	})
	IngestionLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ingestion_latency_seconds",
		Help:    "Event ingestion latency",
		Buckets: prometheus.DefBuckets,
	})
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("request_id", c.GetString("request_id")),
		)
	}
}

func AuthMiddleware(repo APIKeyChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing api key"})
			return
		}
		sourceID, err := repo.ValidateAPIKey(c.Request.Context(), key)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid api key"})
			return
		}
		c.Set("source_id", sourceID)
		c.Next()
	}
}

type APIKeyChecker interface {
	ValidateAPIKey(ctx context.Context, key string) (string, error)
}
