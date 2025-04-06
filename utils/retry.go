package utils

import (
	"fmt"
	"time"

	"gpu-sniper/config"
	"gpu-sniper/ui"
)

// RetryOperation executes the provided function with retry logic
func RetryOperation(operation func() error, retryConfig config.RetryConfig) error {
	var err error
	backoff := retryConfig.InitialBackoff

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		// First attempt or retry
		if attempt > 0 {
			ui.LogWarning("Retry attempt %d of %d after error: %v", 
				attempt, retryConfig.MaxRetries, err)
			
			// Special handling for specific error types
			if err == ErrCaptchaDetected && attempt == 1 {
				ui.LogWarning("CAPTCHA detected, extended cooling period in effect (%v)", backoff)
			}
			
			time.Sleep(backoff)
			
			// Increase backoff for next potential retry
			backoff = time.Duration(float64(backoff) * retryConfig.BackoffFactor)
			if backoff > retryConfig.MaxBackoff {
				backoff = retryConfig.MaxBackoff
			}
		}

		// Execute the operation
		err = operation()
		
		// If successful or specific non-retryable errors, return immediately
		if err == nil {
			return nil
		}
		
		// Check if we've hit max retries
		if attempt == retryConfig.MaxRetries {
			return fmt.Errorf("operation failed after %d attempts: %w", retryConfig.MaxRetries+1, err)
		}
		
		// Check if this error is retryable (if specific errors were provided)
		if len(retryConfig.RetryableErrors) > 0 {
			retryable := false
			for _, retryableErr := range retryConfig.RetryableErrors {
				if err.Error() == retryableErr.Error() {
					retryable = true
					break
				}
			}
			if !retryable {
				return err // Don't retry non-retryable errors
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", retryConfig.MaxRetries+1, err)
}

// Define common errors for use with retry logic
var (
	ErrCaptchaDetected = fmt.Errorf("CAPTCHA challenge detected")
)
