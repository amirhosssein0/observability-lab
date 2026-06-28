package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
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

// initTracer wires up the OpenTelemetry SDK to export spans to Tempo via OTLP/gRPC.
func initTracer(ctx context.Context) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "tempo.tracing.svc.cluster.local:4317"
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(attribute.String("service.name", "observability-lab-app")),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp.Shutdown, nil
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

		span := trace.SpanFromContext(c.Request.Context())
		var traceID string
		if sc := span.SpanContext(); sc.IsValid() {
			traceID = sc.TraceID().String()
		}

		httpRequestsTotal.WithLabelValues(path, strconv.Itoa(status)).Inc()

		observer := httpRequestDuration.WithLabelValues(path)
		if hist, ok := observer.(prometheus.ExemplarObserver); ok && traceID != "" {
			hist.ObserveWithExemplar(duration.Seconds(), prometheus.Labels{"trace_id": traceID})
		} else {
			observer.Observe(duration.Seconds())
		}

		logger.Info("http_request",
			"timestamp", start.UTC().Format(time.RFC3339Nano),
			"method", c.Request.Method,
			"path", path,
			"raw_path", c.Request.URL.Path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"trace_id", traceID,
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
// so Prometheus/Grafana, Loki and Tempo have realistic signal to observe.
func workHandler(c *gin.Context) {
	span := trace.SpanFromContext(c.Request.Context())

	sleepMs := rand.Intn(191) + 10 // 10ms - 200ms
	time.Sleep(time.Duration(sleepMs) * time.Millisecond)

	if rand.Float64() < 0.10 {
		span.RecordError(fmt.Errorf("simulated failure"))
		span.SetStatus(codes.Error, "simulated failure")
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

	ctx := context.Background()
	shutdownTracer, err := initTracer(ctx)
	if err != nil {
		logger.Error("failed to init tracer", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			logger.Error("tracer shutdown error", "error", err)
		}
	}()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("observability-lab-app",
		otelgin.WithFilter(func(req *http.Request) bool {
			switch req.URL.Path {
			case "/health", "/ready", "/metrics":
				return false
			}
			return true
		}),
	))
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