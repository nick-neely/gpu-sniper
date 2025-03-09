package http

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"gpu-sniper/config"
	"gpu-sniper/ui"
	"gpu-sniper/utils"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/113.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
}

// Update the HttpClient definition
var (
	cookieJar, _ = cookiejar.New(nil)
	HttpClient   = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
		Jar: cookieJar,
	}
)

// Add these variables and functions
var (
    relatedURLs = []string{
        "/gp/browse.html?node=193870011&ref_=nav_em__components_0_2_18_7", // PC Components Category
        "/",
        "/support",
        "/deals",
    }
    visitThreshold = 10 // Visit related page every ~10 checks
)

// FetchRetailerPage fetches the HTML content of the retailer page with retry logic
func FetchRetailerPage() (string, error) {
	var responseBody string
	
	operation := func() error {
		req, err := http.NewRequest("GET", config.RetailerURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", getRandomUserAgent())
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")

		ui.LogInfo("Fetching page: %s", config.RetailerURL)
		resp, err := HttpClient.Do(req)
		if err != nil {
			return fmt.Errorf("network error: failed to fetch page. Please check your internet connection: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP error %d: failed to fetch page. This may be due to rate limiting or server-side issues. Please try again later", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		
		responseBody = string(body)
		ui.LogSuccess("Page fetched successfully")
		return nil
	}

	// Execute operation with retry logic
	err := utils.RetryOperation(operation, config.DefaultRetryConfig)
	if err != nil {
		ui.LogError("All retry attempts failed: %v", err)
		return "", err
	}
	
	return responseBody, nil
}

// CreateRequest creates an HTTP request with appropriate headers
func CreateRequest() (*http.Request, error) {
	req, err := http.NewRequest("GET", config.RetailerURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", getRandomUserAgent())
	// Add more realistic browser headers
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	return req, nil
}

func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// VisitRelatedPage with retry logic
func VisitRelatedPage() {
    if rand.Intn(visitThreshold) == 0 {
        baseURL := extractBaseURL(config.RetailerURL)
        if (baseURL != "") {
            randomPath := relatedURLs[rand.Intn(len(relatedURLs))]
            browsePage := baseURL + randomPath
            
            ui.LogInfo("Visiting related page to appear more human: %s", browsePage)
            
            operation := func() error {
                req, _ := http.NewRequest("GET", browsePage, nil)
                req.Header.Set("User-Agent", getRandomUserAgent())
                req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
                
                resp, err := HttpClient.Do(req)
                if err != nil {
                    return fmt.Errorf("network error: failed to visit related page. Please check your internet connection: %w", err)
                }
                defer resp.Body.Close()
                
                if resp.StatusCode != http.StatusOK {
                    return fmt.Errorf("HTTP error %d: failed to load related page. It might be due to rate limiting; check your connection", resp.StatusCode)
                }
                
                // Just discard the body
                return nil
            }
            
            // Execute with retry, but don't fail the whole process if this fails
            _ = utils.RetryOperation(operation, config.RelatedPageRetryConfig)
        }
    }
}

func extractBaseURL(url string) string {
    // Simple extraction - would need more robust parsing in production
    parts := strings.Split(url, "/")
    if len(parts) >= 3 {
        return parts[0] + "//" + parts[2]
    }
    return ""
}
