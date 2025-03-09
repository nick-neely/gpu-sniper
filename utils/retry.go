package utils

import (
	"fmt"
	"time"

	"gpu-sniper/config"
	"gpu-sniper/ui"
)

// RetryOperation executes the provided function with retry logic
func RetryOperation(operation func() error, config config.RetryConfig) error {
	var err error
	backoff := config.InitialBackoff

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// First attempt or retry
		if attempt > 0 {
			ui.LogWarning("Retry attempt %d of %d after error: %v", attempt, config.MaxRetries, err)
			time.Sleep(backoff)
			
			// Increase backoff for next potential retry
			backoff = time.Duration(float64(backoff) * config.BackoffFactor)
			if backoff > config.MaxBackoff {
				backoff = config.MaxBackoff
			}
		}

		// Execute the operation
		err = operation()
		
		// If successful or we've hit max retries, return the result
		if err == nil || attempt == config.MaxRetries {
			return err
		}
		
		// Check if this error is retryable (if specific errors were provided)
		if len(config.RetryableErrors) > 0 {
			retryable := false
			for _, retryableErr := range config.RetryableErrors {
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

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, err)
}
