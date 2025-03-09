package config

import (
	"time"

	"github.com/fatih/color"
)

// Application constants
const (
	// Set the product identifier here (e.g., "B0DVCH9WJH" or "B084BGC5LR")
	ProductID           = "B0DVCH9WJH"
	RetailerURL         = "https://www.amazon.com/gp/product/" + ProductID + "/"
	TargetGPU           = "NVIDIA RTX 5090" // For display purposes; can be updated or removed as needed
	DefaultPollingInterval = 1 * time.Minute
	ProgressWidth       = 40 // Width of the progress bar
)

// Application variables
var (
	PollingInterval = DefaultPollingInterval
	CheckCount      = 0 // Counter for number of checks performed
	LastCheckTime   time.Time // Time of the last check
)

// UI Colors for terminal output
var (
	InfoColor     = color.New(color.FgCyan)
	SuccessColor  = color.New(color.FgGreen)
	ErrorColor    = color.New(color.FgRed)
	WarningColor  = color.New(color.FgYellow)
	HeaderColor   = color.New(color.FgHiWhite, color.Bold)
	TimeColor     = color.New(color.FgHiBlack)
	ProgressColor = color.New(color.FgBlue)
)

// RetryConfig defines the configuration for retry operations
type RetryConfig struct {
	MaxRetries      int           // Maximum number of retry attempts
	InitialBackoff  time.Duration // Initial backoff duration
	MaxBackoff      time.Duration // Maximum backoff duration
	BackoffFactor   float64       // Multiplier for backoff after each retry
	RetryableErrors []error       // Optional specific errors to retry on
}

// Default retry configurations
var (
	DefaultRetryConfig = RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		BackoffFactor:  1.5,
	}

	// Specific retry configurations
	StockCheckRetryConfig = RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
	}

	RelatedPageRetryConfig = RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 300 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
		BackoffFactor:  1.5,
	}
)
