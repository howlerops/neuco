package codegen

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestClassifyError_SentinelMappings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		class     FailureClass
		retryable bool
	}{
		{name: "provider not found", err: ErrProviderNotFound, class: FailureClassConfig, retryable: false},
		{name: "provider not installed", err: ErrProviderNotInstalled, class: FailureClassConfig, retryable: false},
		{name: "config invalid", err: ErrConfigInvalid, class: FailureClassConfig, retryable: false},
		{name: "sandbox provision", err: ErrSandboxProvision, class: FailureClassResource, retryable: true},
		{name: "sandbox timeout", err: ErrSandboxTimeout, class: FailureClassTransient, retryable: true},
		{name: "sandbox destroyed", err: ErrSandboxDestroyed, class: FailureClassResource, retryable: true},
		{name: "validation failed", err: ErrValidationFailed, class: FailureClassValidation, retryable: false},
		{name: "agent execution", err: ErrAgentExecution, class: FailureClassAgent, retryable: true},
		{name: "max retries exceeded", err: ErrMaxRetriesExceeded, class: FailureClassPermanent, retryable: false},
		{name: "api key decryption", err: ErrAPIKeyDecryption, class: FailureClassConfig, retryable: false},
		{name: "unknown defaults transient", err: errors.New("random failure"), class: FailureClassTransient, retryable: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			classified := ClassifyError(fmt.Errorf("wrapped: %w", tc.err))
			if classified == nil {
				t.Fatal("expected classified error, got nil")
			}

			if classified.Class != tc.class {
				t.Fatalf("expected class %q, got %q", tc.class, classified.Class)
			}

			if classified.Retryable != tc.retryable {
				t.Fatalf("expected retryable=%v, got %v", tc.retryable, classified.Retryable)
			}
		})
	}
}

func TestRetryBackoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: -1, want: 5 * time.Second},
		{attempt: 0, want: 5 * time.Second},
		{attempt: 1, want: 10 * time.Second},
		{attempt: 2, want: 20 * time.Second},
		{attempt: 3, want: 40 * time.Second},
		{attempt: 4, want: 80 * time.Second},
		{attempt: 5, want: 2 * time.Minute},
		{attempt: 6, want: 2 * time.Minute},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("attempt_%d", tc.attempt), func(t *testing.T) {
			t.Parallel()

			got := RetryBackoff(tc.attempt)
			if got != tc.want {
				t.Fatalf("RetryBackoff(%d): want %s, got %s", tc.attempt, tc.want, got)
			}
		})
	}
}
