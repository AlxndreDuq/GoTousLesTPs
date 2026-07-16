package main

import (
	"testing"
	"time"
)

func TestEnvOr(t *testing.T) {
	t.Setenv("MIRA_TEST_ENV_OR", "value")
	if got := envOr("MIRA_TEST_ENV_OR", "fallback"); got != "value" {
		t.Fatalf("envOr() = %q, want %q", got, "value")
	}
	if got := envOr("MIRA_TEST_ENV_OR_UNSET", "fallback"); got != "fallback" {
		t.Fatalf("envOr() = %q, want fallback %q", got, "fallback")
	}
}

func TestEnvIntOr(t *testing.T) {
	t.Setenv("MIRA_TEST_ENV_INT", "42")
	if got := envIntOr("MIRA_TEST_ENV_INT", 4); got != 42 {
		t.Fatalf("envIntOr() = %d, want 42", got)
	}
	if got := envIntOr("MIRA_TEST_ENV_INT_UNSET", 4); got != 4 {
		t.Fatalf("envIntOr() = %d, want fallback 4", got)
	}

	t.Setenv("MIRA_TEST_ENV_INT_BAD", "not-a-number")
	if got := envIntOr("MIRA_TEST_ENV_INT_BAD", 7); got != 7 {
		t.Fatalf("envIntOr() with invalid value = %d, want fallback 7", got)
	}
}

func TestEnvDurationOr(t *testing.T) {
	t.Setenv("MIRA_TEST_ENV_DURATION", "30s")
	if got := envDurationOr("MIRA_TEST_ENV_DURATION", time.Second); got != 30*time.Second {
		t.Fatalf("envDurationOr() = %v, want 30s", got)
	}
	if got := envDurationOr("MIRA_TEST_ENV_DURATION_UNSET", 5*time.Second); got != 5*time.Second {
		t.Fatalf("envDurationOr() = %v, want fallback 5s", got)
	}

	t.Setenv("MIRA_TEST_ENV_DURATION_BAD", "not-a-duration")
	if got := envDurationOr("MIRA_TEST_ENV_DURATION_BAD", 5*time.Second); got != 5*time.Second {
		t.Fatalf("envDurationOr() with invalid value = %v, want fallback 5s", got)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	for _, key := range []string{"PORT", "DATABASE_URL", "ENRICHMENT_WORKERS", "ENRICHMENT_QUEUE_SIZE", "ENRICHMENT_TIMEOUT"} {
		t.Setenv(key, "")
	}

	cfg := loadConfig()

	if cfg.port != "8080" {
		t.Errorf("port = %q, want %q", cfg.port, "8080")
	}
	if cfg.enrichmentWorkers != 4 {
		t.Errorf("enrichmentWorkers = %d, want 4", cfg.enrichmentWorkers)
	}
	if cfg.enrichmentQueueSize != 100 {
		t.Errorf("enrichmentQueueSize = %d, want 100", cfg.enrichmentQueueSize)
	}
	if cfg.enrichmentTimeout != 10*time.Second {
		t.Errorf("enrichmentTimeout = %v, want 10s", cfg.enrichmentTimeout)
	}
}
