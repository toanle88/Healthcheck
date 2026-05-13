package monitor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	Meter            metric.Meter
	CheckCounter     metric.Int64Counter
	LatencyHistogram metric.Float64Histogram
)

// InitOTel initializes the OpenTelemetry SDK.
// It returns a metrics handler, a shutdown function, and an error.
func InitOTel(ctx context.Context, serviceName string) (http.Handler, func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 1. Setup Tracing (OTLP/HTTP exporter for Jaeger)
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("jaeger:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// 2. Setup Metrics
	metricExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Prometheus exporter (acts as a Reader for OTel)
	// We link it directly to the DefaultRegisterer so it shows up in /metrics
	promExporter, err := otelprom.New(otelprom.WithRegisterer(prometheus.DefaultRegisterer))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(1*time.Minute))),
		sdkmetric.WithReader(promExporter),
	)
	otel.SetMeterProvider(mp)

	// Create the Meter and instruments
	Meter = mp.Meter(serviceName)
	CheckCounter, _ = Meter.Int64Counter("healthcheck_status_total",
		metric.WithDescription("Total number of health checks performed"))
	LatencyHistogram, _ = Meter.Float64Histogram("healthcheck_latency_seconds",
		metric.WithDescription("Latency of health checks in seconds"))

	// Create a dedicated handler for the prometheus metrics
	// This avoids the global registry conflict
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})

	// Return a combined shutdown function
	return handler, func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
		if err := mp.Shutdown(ctx); err != nil {
			return err
		}
		return nil
	}, nil
}
