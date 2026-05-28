package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCLI_UnsupportedProvider(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-provider", "unknown_provider")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code for unsupported provider")
	}
	outStr := string(output)
	if !strings.Contains(outStr, "unsupported provider") {
		t.Errorf("expected error message to contain 'unsupported provider', got: %s", outStr)
	}
}

func TestCLI_InvalidConcurrency(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-concurrency", "0")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit code for concurrency <= 0")
	}
	outStr := string(output)
	if !strings.Contains(outStr, "concurrency must be greater than zero") {
		t.Errorf("expected error message about concurrency, got: %s", outStr)
	}
}



func TestCLI_HelpFlag(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-h")
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "Usage") {
		t.Errorf("expected usage output, got: %s", string(out))
	}
}


func TestCLI_ValidConcurrencyFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
}

