package stock

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"gpu-sniper/config"
	httpClient "gpu-sniper/http"
	"gpu-sniper/ui"
	"gpu-sniper/utils"
)

// Global variable to track the current progress tracker
var CurrentProgressTracker *ui.ProgressTracker

// ParseStockStatus parses an HTTP response to check if the product is in stock
func ParseStockStatus(resp *http.Response) (bool, error) {
	// Parse directly from response body
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if (err != nil) {
		return false, fmt.Errorf("error parsing HTML: %w", err)
	}
	inStock := doc.Find("#add-to-cart-button").Length() > 0
	return inStock, nil
}

func AdjustPollingByTimeOfDay() time.Duration {
    hour := time.Now().Hour()
    
    // Late night hours (fewer checks needed)
    if hour >= 1 && hour < 6 {
        return config.PollingInterval * 2
    }
    
    // High traffic hours (be more cautious)
    if (hour >= 12 && hour <= 14) || (hour >= 18 && hour <= 21) {
        return config.PollingInterval + time.Duration(rand.Int63n(int64(30*time.Second)))
    }
    
    return config.PollingInterval
}

func GetNextPollingInterval() time.Duration {
    baseInterval := AdjustPollingByTimeOfDay()
    jitter := time.Duration(rand.Int63n(int64(5 * time.Second)))
    return baseInterval + jitter
}

// SetProgressTracker updates the current progress tracker
func SetProgressTracker(tracker *ui.ProgressTracker) {
	CurrentProgressTracker = tracker
}

// UpdatePollingInterval updates the current polling interval and progress tracker if available
func UpdatePollingInterval(newInterval time.Duration) {
    config.PollingInterval = newInterval
    
    // If we have an active progress tracker, update it
    if CurrentProgressTracker != nil {
        CurrentProgressTracker.UpdateDuration(newInterval)
        ui.LogInfo("Polling interval updated to %v due to rate limiting", newInterval)
    }
}

// CheckStock performs a stock check and analyzes the results with retry logic
func CheckStock() bool {
	// Update check counter
	config.CheckCount++
	config.LastCheckTime = time.Now()
	
	// Update status in the tracker if available
	if CurrentProgressTracker != nil {
		CurrentProgressTracker.UpdateStatus("Checking")
	}
	
	httpClient.VisitRelatedPage()
	
	// Clear the progress bar line and print header
	fmt.Printf("\r\033[K")
	config.HeaderColor.Printf("\n[STOCK CHECK #%d] %s\n", config.CheckCount, time.Now().Format("2006-01-02 03:04:05 PM"))
	fmt.Println(strings.Repeat("─", 50))

	var inStock bool

	operation := func() error {
		// Create and send HTTP request
		req, err := httpClient.CreateRequest()
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		ui.LogInfo("Fetching page: %s", config.RetailerURL)
		resp, err := httpClient.HttpClient.Do(req)
		if err != nil {
			return fmt.Errorf("network error: failed to fetch page. Please check your internet connection: %w", err)
		}
		defer resp.Body.Close()

		// Updated error messages for HTTP failures
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
			ui.LogWarning("HTTP %d received, indicating rate limiting. Please wait and check your connection.", resp.StatusCode)
			// Modify the polling interval temporarily
			newInterval := config.PollingInterval * 2
			if newInterval > 5*time.Minute {
				newInterval = 5 * time.Minute
			}
			UpdatePollingInterval(newInterval)
			
			if CurrentProgressTracker != nil {
				CurrentProgressTracker.UpdateStatus("Rate limited")
			}
			
			return fmt.Errorf("HTTP %d: rate limiting in effect. Too many requests; please try again later", resp.StatusCode)
		} else if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP error %d: failed to fetch the product page. Verify your network connection or check if the website is experiencing issues", resp.StatusCode)
		} else {
			// Gradually reset polling interval on successful requests
			if config.PollingInterval > config.DefaultPollingInterval {
				newInterval := time.Duration(float64(config.PollingInterval) * 0.8)
				if newInterval < config.DefaultPollingInterval {
					newInterval = config.DefaultPollingInterval
				}
				UpdatePollingInterval(newInterval)
			}
		}
		ui.LogSuccess("Page fetched successfully")

		// Parse the response directly
		ui.LogInfo("Analyzing product availability...")
		stockStatus, err := ParseStockStatus(resp)
		if err != nil {
			return fmt.Errorf("failed to parse product page: %w", err)
		}
		
		inStock = stockStatus
		
		// After successful check, update status
		if CurrentProgressTracker != nil {
			CurrentProgressTracker.UpdateStatus("Waiting")
		}
		
		return nil
	}
	
	// Execute with retry logic
	err := utils.RetryOperation(operation, config.StockCheckRetryConfig)
	
	fmt.Println(strings.Repeat("─", 50))
	
	if err != nil {
		ui.LogError("Stock check failed after retries: %v", err)
		return false
	}
	
	// Reset status to waiting if we have a tracker
	if CurrentProgressTracker != nil {
		CurrentProgressTracker.UpdateStatus("Waiting")
	}

	if inStock {
		config.SuccessColor.Printf("✓ %s is IN STOCK!\n", config.TargetGPU)
		return true
	} else {
		config.ErrorColor.Printf("✗ %s is not in stock\n", config.TargetGPU)
		return false
	}
}
