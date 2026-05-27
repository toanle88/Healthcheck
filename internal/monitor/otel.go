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
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
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

func init() {
	// Initialize with No-Ops to prevent nil panics if InitOTel fails
	nm := noop.NewMeterProvider().Meter("noop")
	Meter = nm
	CheckCounter, _ = nm.Int64Counter("noop_total")
	LatencyHistogram, _ = nm.Float64Histogram("noop_latency")
}

// setupTracing configures the trace exporter and trace provider, connecting to Jaeger locally or Azure App Insights.
func setupTracing(ctx context.Context, res *resource.Resource, isAzure bool, connString string) (*trace.TracerProvider, error) {
	traceOpts := []otlptracehttp.Option{}
	if isAzure {
		host, ikey := parseConnectionString(connString)
		traceOpts = append(traceOpts,
			otlptracehttp.WithEndpoint(host),
			otlptracehttp.WithURLPath("/v2.1/otlp/v1/traces"),
			otlptracehttp.WithHeaders(map[string]string{"x-otlp-api-key": ikey}),
		)
	} else {
		traceOpts = append(traceOpts, otlptracehttp.WithEndpoint("jaeger:4318"), otlptracehttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	return tp, nil
}

// setupMetrics configures OTel metrics using periodic OTLP HTTP metrics exporter (for Azure) or Prometheus exporter.
func setupMetrics(ctx context.Context, res *resource.Resource, isAzure bool, connString string, promExporter *otelprom.Exporter) (*sdkmetric.MeterProvider, error) {
	var mp *sdkmetric.MeterProvider
	if isAzure {
		host, ikey := parseConnectionString(connString)
		metricExporter, err := otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(host),
			otlpmetrichttp.WithURLPath("/v2.1/otlp/v1/metrics"),
			otlpmetrichttp.WithHeaders(map[string]string{"x-otlp-api-key": ikey}),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create metric exporter: %w", err)
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
	return mp, nil
}

// InitOTel initializes the OpenTelemetry SDK with resource detection, trace providers, and metrics meters.
// Returns an HTTP handler for exposing Prometheus metrics, a shutdown function, and any initialization error.
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
	env := os.Getenv("ENV")
	isLocal := env == "local" || env == "development" || env == ""
	isAzure := connString != "" && !isLocal

	// 1. Setup Tracing
	tp, err := setupTracing(ctx, res, isAzure, connString)
	if err != nil {
		return nil, nil, err
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// 2. Setup Metrics
	promExporter, _ := otelprom.New(otelprom.WithRegisterer(prometheus.DefaultRegisterer))
	mp, err := setupMetrics(ctx, res, isAzure, connString, promExporter)
	if err != nil {
		return nil, nil, err
	}
	otel.SetMeterProvider(mp)

	// Initialize real instruments
	Meter = mp.Meter(serviceName)
	CheckCounter, _ = Meter.Int64Counter("healthcheck_status_total",
		metric.WithDescription("Total number of health checks performed"))
	LatencyHistogram, _ = Meter.Float64Histogram("healthcheck_latency_seconds",
		metric.WithDescription("Latency of health checks in seconds"))

	// Create a dedicated handler for the prometheus metrics
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})

	// Return a combined shutdown function
	return handler, func(ctx context.Context) error {
		var errs []error
		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		if mp != nil {
			if err := mp.Shutdown(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("otel shutdown error(s): %v", errs)
		}
		return nil
	}, nil
}

// parseConnectionString parses an Azure Application Insights connection string
// and extracts the target ingestion endpoint host and the instrumentation key.
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

	// Transform classic App Insights host to modern Monitor Ingestion host if needed
	// e.g. eastasia-0.in.applicationinsights.azure.com -> eastasia.ingestion.monitor.azure.com
	if strings.Contains(endpoint, "applicationinsights.azure.com") {
		regionParts := strings.Split(endpoint, ".")
		if len(regionParts) > 0 {
			region := strings.Split(regionParts[0], "-")[0]
			endpoint = fmt.Sprintf("%s.ingestion.monitor.azure.com", region)
		}
	}

	return endpoint, ikey
}
