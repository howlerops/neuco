package codegen

import (
	"errors"
	"time"
)

var (
	ErrProviderNotFound      = errors.New("provider not found")
	ErrProviderNotInstalled  = errors.New("provider not installed")
	ErrConfigInvalid         = errors.New("invalid provider config")
	ErrSandboxProvision      = errors.New("sandbox provision failed")
	ErrSandboxTimeout        = errors.New("sandbox timeout")
	ErrSandboxDestroyed      = errors.New("sandbox destroyed")
	ErrValidationFailed      = errors.New("validation failed")
	ErrAgentExecution        = errors.New("agent execution failed")
	ErrMaxRetriesExceeded    = errors.New("max retries exceeded")
	ErrAPIKeyDecryption      = errors.New("api key decryption failed")
)

// FailureClass describes the broad category of an execution failure.
type FailureClass string

const (
	FailureClassTransient  FailureClass = "transient"
	FailureClassValidation FailureClass = "validation"
	FailureClassConfig     FailureClass = "config"
	FailureClassResource   FailureClass = "resource"
	FailureClassAgent      FailureClass = "agent"
	FailureClassPermanent  FailureClass = "permanent"
)

// ClassifiedError wraps an error with failure metadata used for retry policy.
type ClassifiedError struct {
	Err       error
	Class     FailureClass
	Retryable bool
	Message   string
}

func (e *ClassifiedError) Error() string {
	if e == nil {
		return ""
	}

	if e.Message == "" {
		if e.Err == nil {
			return ""
		}
		return e.Err.Error()
	}

	if e.Err == nil {
		return e.Message
	}

	return e.Message + ": " + e.Err.Error()
}

func (e *ClassifiedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ClassifyError maps known sentinel errors to failure classes and retry behavior.
func ClassifyError(err error) *ClassifiedError {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrProviderNotFound):
		return &ClassifiedError{Err: err, Class: FailureClassConfig, Retryable: false}
	case errors.Is(err, ErrProviderNotInstalled):
		return &ClassifiedError{Err: err, Class: FailureClassConfig, Retryable: false}
	case errors.Is(err, ErrConfigInvalid):
		return &ClassifiedError{Err: err, Class: FailureClassConfig, Retryable: false}
	case errors.Is(err, ErrSandboxProvision):
		return &ClassifiedError{Err: err, Class: FailureClassResource, Retryable: true}
	case errors.Is(err, ErrSandboxTimeout):
		return &ClassifiedError{Err: err, Class: FailureClassTransient, Retryable: true}
	case errors.Is(err, ErrSandboxDestroyed):
		return &ClassifiedError{Err: err, Class: FailureClassResource, Retryable: true}
	case errors.Is(err, ErrValidationFailed):
		return &ClassifiedError{Err: err, Class: FailureClassValidation, Retryable: false}
	case errors.Is(err, ErrAgentExecution):
		return &ClassifiedError{Err: err, Class: FailureClassAgent, Retryable: true}
	case errors.Is(err, ErrMaxRetriesExceeded):
		return &ClassifiedError{Err: err, Class: FailureClassPermanent, Retryable: false}
	case errors.Is(err, ErrAPIKeyDecryption):
		return &ClassifiedError{Err: err, Class: FailureClassConfig, Retryable: false}
	default:
		return &ClassifiedError{Err: err, Class: FailureClassTransient, Retryable: true}
	}
}

// RetryBackoff returns exponential backoff delay for a 0-indexed retry attempt.
func RetryBackoff(attempt int) time.Duration {
	base := 5 * time.Second
	max := 2 * time.Minute

	if attempt <= 0 {
		return base
	}

	delay := base
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay >= max {
			return max
		}
	}

	return delay
}
