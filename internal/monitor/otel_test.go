package monitor

import (
	"context"
	"os"
	"testing"
)

func TestParseConnectionString(t *testing.T) {
	tests := []struct {
		connStr          string
		expectedEndpoint string
		expectedIkey     string
	}{
		{
			"InstrumentationKey=123-456;IngestionEndpoint=https://eastasia.applicationinsights.azure.com/",
			"eastasia.ingestion.monitor.azure.com",
			"123-456",
		},
		{
			"InstrumentationKey=abc;IngestionEndpoint=https://eastasia-0.in.applicationinsights.azure.com/",
			"eastasia.ingestion.monitor.azure.com",
			"abc",
		},
	}

	for _, tc := range tests {
		endpoint, ikey := parseConnectionString(tc.connStr)
		if endpoint != tc.expectedEndpoint {
			t.Errorf("expected endpoint %s, got %s", tc.expectedEndpoint, endpoint)
		}
		if ikey != tc.expectedIkey {
			t.Errorf("expected ikey %s, got %s", tc.expectedIkey, ikey)
		}
	}
}

func TestInitOTelLocal(t *testing.T) {
	os.Setenv("ENV", "local")
	defer os.Unsetenv("ENV")

	ctx := context.Background()
	handler, shutdown, err := InitOTel(ctx, "test-service")
	if err != nil {
		t.Fatalf("expected no error from InitOTel in local mode, got %v", err)
	}

	if handler == nil {
		t.Errorf("expected metrics handler to be initialized, got nil")
	}

	if shutdown == nil {
		t.Errorf("expected shutdown function, got nil")
	} else {
		err := shutdown(ctx)
		if err != nil {
			t.Errorf("expected no error shutting down OTel: %v", err)
		}
	}
}
