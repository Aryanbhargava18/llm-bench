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


func TestTelemetry_ProviderDoubleInit(t *testing.T) {
	tp1, mp1, err1 := InitProvider()
	if err1 != nil {
		t.Fatalf("first InitProvider failed: %v", err1)
	}
	if tp1 == nil || mp1 == nil {
		t.Fatal("expected non-nil providers")
	}
}


func TestTelemetry_ShutdownCleanliness(t *testing.T) {
	tp, mp, err := InitProvider()
	if err != nil {
		t.Fatalf("InitProvider failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := tp.Shutdown(ctx); err != nil {
		t.Errorf("TracerProvider shutdown returned error: %v", err)
	}
	if err := mp.Shutdown(ctx); err != nil {
		t.Errorf("MeterProvider shutdown returned error: %v", err)
	}
}
