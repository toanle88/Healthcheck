package monitor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
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

	connString := os.Getenv("APPLICATIONINSIGHTS_CONNECTION_STRING")
	isAzure := connString != "" && os.Getenv("ENV") != "local"

	// 1. Setup Tracing
	traceOpts := []otlptracehttp.Option{}
	if isAzure {
		endpoint, ikey := parseConnectionString(connString)
		traceOpts = append(traceOpts,
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithHeaders(map[string]string{"x-otlp-api-key": ikey}),
		)
	} else {
		traceOpts = append(traceOpts, otlptracehttp.WithEndpoint("jaeger:4318"), otlptracehttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// 2. Setup Metrics
	var mp *sdkmetric.MeterProvider
	promExporter, _ := otelprom.New(otelprom.WithRegisterer(prometheus.DefaultRegisterer))

	if isAzure {
		endpoint, ikey := parseConnectionString(connString)
		metricExporter, err := otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(endpoint),
			otlpmetrichttp.WithHeaders(map[string]string{"x-otlp-api-key": ikey}),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create metric exporter: %w", err)
		}

		mp = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(1*time.Minute))),
			sdkmetric.WithReader(promExporter),
		)
	} else {
		mp = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(promExporter),
		)
	}
	otel.SetMeterProvider(mp)

	// Create the Meter and instruments
	Meter = mp.Meter(serviceName)
	CheckCounter, _ = Meter.Int64Counter("healthcheck_status_total",
		metric.WithDescription("Total number of health checks performed"))
	LatencyHistogram, _ = Meter.Float64Histogram("healthcheck_latency_seconds",
		metric.WithDescription("Latency of health checks in seconds"))

	// Create a dedicated handler for the prometheus metrics
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})

	// Return a combined shutdown function
	return handler, func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
		if mp != nil {
			if err := mp.Shutdown(ctx); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

// parseConnectionString extracts the ingestion endpoint and instrumentation key.
// Azure connection string looks like: InstrumentationKey=...;IngestionEndpoint=https://.../
func parseConnectionString(connStr string) (string, string) {
	parts := strings.Split(connStr, ";")
	var ikey, endpoint string
	for _, p := range parts {
		if strings.HasPrefix(p, "InstrumentationKey=") {
			ikey = strings.TrimPrefix(p, "InstrumentationKey=")
		}
		if strings.HasPrefix(p, "IngestionEndpoint=") {
			endpoint = strings.TrimPrefix(p, "IngestionEndpoint=")
			endpoint = strings.TrimPrefix(endpoint, "https://")
			endpoint = strings.TrimSuffix(endpoint, "/")
		}
	}
	// Add the OTLP path suffix if missing (Azure expects /v2.1/otlp)
	if !strings.Contains(endpoint, "/v2.1/otlp") {
		endpoint = endpoint + "/v2.1/otlp"
	}
	return endpoint, ikey
}
