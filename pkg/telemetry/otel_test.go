package telemetry

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

func TestInitProvider(t *testing.T) {
	origTP := otel.GetTracerProvider()
	origMP := otel.GetMeterProvider()
	origProp := otel.GetTextMapPropagator()

	t.Cleanup(func() {
		otel.SetTracerProvider(origTP)
		otel.SetMeterProvider(origMP)
		otel.SetTextMapPropagator(origProp)
	})

	tp, mp, err := InitProvider()
	if err != nil {
		t.Fatalf("InitProvider failed: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil TracerProvider")
	}
	if mp == nil {
		t.Fatal("expected non-nil MeterProvider")
	}

	// Verify global providers were configured
	if otel.GetTracerProvider() != tp {
		t.Error("global TracerProvider was not set to the initialized provider")
	}
	if otel.GetMeterProvider() != mp {
		t.Error("global MeterProvider was not set to the initialized provider")
	}

	// Test clean shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := Shutdown(ctx, tp, mp); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestShutdown_NilProviders(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Should not panic or return error when providers are nil
	if err := Shutdown(ctx, nil, nil); err != nil {
		t.Errorf("expected no error for nil providers, got %v", err)
	}
}
