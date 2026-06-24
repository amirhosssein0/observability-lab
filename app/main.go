package main

import (
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests, partitioned by path and status code.",
		},
		[]string{"path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds, partitioned by path.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.15, 0.2, 0.25, 0.5, 1.0},
		},
		[]string{"path"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

// observabilityMiddleware records metrics and structured logs for every request.
// Only routes registered with Gin (c.FullPath() non-empty) are labeled by their
// route pattern; anything else (404s, unknown paths) is bucketed under "unmatched"
// to avoid unbounded label cardinality in Prometheus.
func observabilityMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		duration := time.Since(start)

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		status := c.Writer.Status()

		httpRequestsTotal.WithLabelValues(path, strconv.Itoa(status)).Inc()
		httpRequestDuration.WithLabelValues(path).Observe(duration.Seconds())

		logger.Info("http_request",
			"timestamp", start.UTC().Format(time.RFC3339Nano),
			"method", c.Request.Method,
			"path", path,
			"raw_path", c.Request.URL.Path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
		)
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "up"})
}

func readyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// workHandler simulates variable-latency work with an occasional failure,
// so Prometheus/Grafana and Loki have realistic signal to observe.
func workHandler(c *gin.Context) {
	sleepMs := rand.Intn(191) + 10 // 10ms - 200ms
	time.Sleep(time.Duration(sleepMs) * time.Millisecond)

	if rand.Float64() < 0.10 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "simulated failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "done", "duration_ms": sleepMs})
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(observabilityMiddleware(logger))

	r.GET("/health", healthHandler)
	r.GET("/ready", readyHandler)
	r.GET("/work", workHandler)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	logger.Info("server starting", "addr", ":8080")
	if err := r.Run(":8080"); err != nil {
		logger.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}