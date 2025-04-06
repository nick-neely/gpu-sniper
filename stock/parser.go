package stock

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
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

// Custom error type for CAPTCHA detection
var ErrCaptchaDetected = fmt.Errorf("CAPTCHA challenge detected")

// ParseStockStatus parses an HTTP response to check if the product is in stock
func ParseStockStatus(resp *http.Response) (bool, error) {
    // Parse directly from response body
    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if (err != nil) {
        return false, fmt.Errorf("error parsing HTML: %w", err)
    }
    
    // Primary check - exact ID match
    addToCartButton := doc.Find("#add-to-cart-button")
    if addToCartButton.Length() > 0 {
        // Check if button is not disabled (some sites have disabled buttons when out of stock)
        _, disabled := addToCartButton.Attr("disabled")
        if !disabled {
            ui.LogInfo("Found enabled add-to-cart button with ID 'add-to-cart-button'")
            return true, nil
        } else {
            ui.LogInfo("Add-to-cart button found but is disabled")
        }
    }
    
    // Fallback checks for other common patterns
    selectors := []string{
        "[id*=add-to-cart]", 
        "[class*=add-to-cart]",
        "[id*=addToCart]",
        "[class*=addToCart]",
        ".btn-add-to-cart:not([disabled])",
        "button:contains('Add to Cart')",
        "input[type=submit][value*='Add to Cart']",
    }
    
    for _, selector := range selectors {
        elements := doc.Find(selector)
        if elements.Length() > 0 {
            ui.LogInfo("Found alternative add-to-cart element with selector: %s", selector)
            return true, nil
        }
    }
    
    // Additional check for "Out of Stock" text which indicates item exists but is unavailable
    outOfStockTexts := []string{"Out of Stock", "Sold Out", "Currently unavailable", "Temporarily out of stock"}
    for _, text := range outOfStockTexts {
        if doc.Find(fmt.Sprintf("*:contains('%s')", text)).Length() > 0 {
            ui.LogInfo("Page contains '%s' text, confirming item exists but is out of stock", text)
            return false, nil
        }
    }
    
    ui.LogInfo("No add-to-cart indicators found on the page")
    return false, nil
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
	var captchaDetected bool

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

		if isCaptchaPage(resp) {
            ui.LogWarning("CAPTCHA detected - increasing delay and implementing cooling period")
            
            // Apply exponential backoff to polling interval
            newInterval := config.PollingInterval * 3
            if newInterval > 15*time.Minute {
                newInterval = 15*time.Minute
            }
            UpdatePollingInterval(newInterval)
            
            // Update progress tracker
            if CurrentProgressTracker != nil {
                CurrentProgressTracker.UpdateStatus("Cooling Down - CAPTCHA detected")
            }
            
            captchaDetected = true
            return ErrCaptchaDetected
        }

		DebugSaveHTML(resp)

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
	
	var err error
	// Choose the appropriate retry configuration based on the situation
	if captchaDetected {
		// Use CaptchaRetryConfig for CAPTCHA-related retries
		err = utils.RetryOperation(operation, config.CaptchaRetryConfig)
	} else {
		// Use StockCheckRetryConfig for normal stock checks
		err = utils.RetryOperation(operation, config.StockCheckRetryConfig)
	}
	
	fmt.Println(strings.Repeat("─", 50))
	
	if err == ErrCaptchaDetected {
		ui.LogWarning("CAPTCHA detected, cooling down for an extended period")
		return false
	}

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

func DebugSaveHTML(resp *http.Response) error {
    // Clone the body since reading it consumes it
    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }
    
    // Restore the body for other functions
    resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
    
    // Save HTML to file
    filename := fmt.Sprintf("debug_%s.html", time.Now().Format("20060102_150405"))
    err = os.WriteFile(filename, bodyBytes, 0644)
    if err != nil {
        return err
    }
    
    ui.LogInfo("Saved HTML content to %s for debugging", filename)
    return nil
}

func isCaptchaPage(resp *http.Response) bool {
    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return false
    }
    
    // Restore the body for other operations
    resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
    
    // Check for CAPTCHA indicators in the response body
    bodyStr := string(bodyBytes)
    captchaIndicators := []string{
        "captcha", "robot", "characters you see", "verify human", 
        "Enter the characters", "validateCaptcha",
    }
    
    for _, indicator := range captchaIndicators {
        if strings.Contains(strings.ToLower(bodyStr), strings.ToLower(indicator)) {
            return true
        }
    }
    
    return false
}
