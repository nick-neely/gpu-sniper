package http

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
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

// New variables for session persistence
var (
	CurrentUserAgent string
	cookieFile       = "cookies.json"
	cookieMutex      sync.Mutex
	cookieChan       = make(chan map[string][]*http.Cookie, 1) // Buffered channel
)

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

// Update relatedURLs with more Amazon-specific paths
var (
    relatedURLs = []string{
        "/gp/browse.html?node=193870011", // PC Components
        "/gp/browse.html?node=172282", // Electronics
        "/gp/browse.html?node=17923671011", // Amazon basics
        "/gp/bestsellers/", // Best Sellers
        "/gp/new-releases/", // New Releases
        "/gp/goldbox", // Today's Deals
        "/", // Homepage
        "/gp/help/customer/display.html", // Help
    }
    visitThreshold = 8 // Visit related page every ~8 checks
)

// New init function for session persistence
func init() {
	CurrentUserAgent = userAgents[rand.Intn(len(userAgents))]
	loadCookiesFromFile()
	go periodicCookieSave()
	go cookieSaver() // Start the cookie saving goroutine
}

// Load cookies from file and set them in the cookie jar
func loadCookiesFromFile() {
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		return
	}
	var store map[string][]*http.Cookie
	if err = json.Unmarshal(data, &store); err != nil {
		return
	}
	// Load cookies for each domain
	for domain, cookies := range store {
		u := &url.URL{Scheme: "https", Host: domain}
		cookieJar.SetCookies(u, cookies)
	}
}

// Save cookies for the retailer domain to file
func saveCookiesToFile() {
	u, err := url.Parse(config.RetailerURL)
	if err != nil {
		return
	}
	cookies := cookieJar.Cookies(u)
	store := map[string][]*http.Cookie{
		u.Host: cookies,
	}
	// Send cookies to the channel for saving
	select {
	case cookieChan <- store:
		// Successfully sent
	default:
		// Channel is full; don't block
		ui.LogWarning("Cookie save channel is full, skipping save")
	}
}

// Cookie saver goroutine
func cookieSaver() {
	for cookies := range cookieChan {
		cookieMutex.Lock()
		data, err := json.Marshal(cookies)
		cookieMutex.Unlock()
		if err != nil {
			ui.LogError("Error marshaling cookies: %v", err)
			continue
		}
		err = os.WriteFile(cookieFile, data, 0644)
		if err != nil {
			ui.LogError("Error writing cookie file: %v", err)
		}
	}
}

// Periodically save cookies every 30 seconds
func periodicCookieSave() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		saveCookiesToFile()
	}
}

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

	// Execute operation with retry logic - use DefaultRetryConfig since this is a standard fetch
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
    req.Header.Set("Accept-Language", "en-US,en;q=0.8")
    req.Header.Set("Accept-Encoding", "gzip, deflate, br")
    req.Header.Set("DNT", "1")
    req.Header.Set("Connection", "keep-alive")
    req.Header.Set("Sec-Fetch-Dest", "document")
    req.Header.Set("Sec-Fetch-Mode", "navigate")
    req.Header.Set("Sec-Fetch-Site", "none")
    req.Header.Set("Sec-Fetch-User", "?1")
    req.Header.Set("Upgrade-Insecure-Requests", "1")
    req.Header.Set("Cache-Control", "max-age=0")
	return req, nil
}

// Modify getRandomUserAgent to return a consistent session agent
func getRandomUserAgent() string {
	return CurrentUserAgent
}

// Enhanced VisitRelatedPage function with more natural browsing behavior
func VisitRelatedPage() {
    if rand.Intn(visitThreshold) != 0 {
        return // Don't visit every time
    }
    
    baseURL := extractBaseURL(config.RetailerURL)
    if baseURL == "" {
        return
    }
    
    // Determine how many related pages to visit (0-2)
    pagesToVisit := rand.Intn(3)
    if pagesToVisit == 0 {
        return // Sometimes don't browse at all
    }
    
    ui.LogInfo("Browsing %d related page(s) to appear more human", pagesToVisit)
    
    for i := 0; i < pagesToVisit; i++ {
        // Get random related page
        randomPath := relatedURLs[rand.Intn(len(relatedURLs))]
        browsePage := baseURL + randomPath
        
        // Add random delay between page visits (1-3 seconds)
        time.Sleep(time.Duration(1000+rand.Intn(2000)) * time.Millisecond)
        
        // Visit the related page
        req, err := http.NewRequest("GET", browsePage, nil)
        if err != nil {
            continue
        }
        
        // Use consistent headers for the session
        req.Header.Set("User-Agent", CurrentUserAgent)
        req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
        req.Header.Set("Accept-Language", "en-US,en;q=0.8")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("DNT", "1")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Cache-Control", "max-age=0")
        
        if i > 0 {
            // Add referrer after first page to look natural
            req.Header.Set("Referer", baseURL)
        }
        
        resp, err := HttpClient.Do(req)
        if err != nil {
            continue
        }
        
        // Read the body content
        body, err := io.ReadAll(resp.Body)
        resp.Body.Close()
        
        if err == nil {
            // Sometimes follow internal links (25% chance)
            if rand.Intn(4) == 0 && len(body) > 0 {
                internalLink := extractRandomInternalLink(string(body), baseURL)
                if internalLink != "" && internalLink != browsePage {
                    // Add a more natural delay before clicking internal link (1.5-4.5 seconds)
                    time.Sleep(time.Duration(1500+rand.Intn(3000)) * time.Millisecond)
                    
                    // Visit internal link with retry logic
                    internalOperation := func() error {
                        internalReq, _ := http.NewRequest("GET", internalLink, nil)
                        internalReq.Header.Set("User-Agent", CurrentUserAgent)
                        internalReq.Header.Set("Referer", browsePage)
                        internalResp, err := HttpClient.Do(internalReq)
                        
                        if err != nil {
                            return fmt.Errorf("failed to fetch internal link: %w", err)
                        }
                        
                        // Discard the response body but close it properly
                        io.Copy(io.Discard, internalResp.Body)
                        internalResp.Body.Close()
                        return nil
                    }
                    
                    // Use RelatedPageRetryConfig for internal link navigation
                    _ = utils.RetryOperation(internalOperation, config.RelatedPageRetryConfig)
                }
            }
        }
    }
}

// extractRandomInternalLink extracts a random internal Amazon link from HTML content
func extractRandomInternalLink(htmlContent, baseURL string) string {
    // Define patterns for Amazon internal links
    patterns := []string{
        `href="(/dp/[A-Z0-9]{10}[^"]*)"`,           // Product links
        `href="(/gp/product/[A-Z0-9]{10}[^"]*)"`,   // Alternative product links
        `href="(/gp/browse\.html\?node=[0-9]+[^"]*)"`, // Category browsing
        `href="(/s\?k=[^"&]+)"`,                    // Search results
    }
    
    var allMatches []string
    
    // Find all matches for each pattern
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        matches := re.FindAllStringSubmatch(htmlContent, -1)
        
        for _, match := range matches {
            if len(match) >= 2 {
                // Add the match to our collection
                allMatches = append(allMatches, match[1])
            }
        }
        
        // Limit the number of matches to prevent excessive memory usage
        if len(allMatches) > 50 {
            break
        }
    }
    
    // If we found links, return a random one
    if len(allMatches) > 0 {
        randomLink := allMatches[rand.Intn(len(allMatches))]
        // Ensure the link is an absolute URL
        if strings.HasPrefix(randomLink, "/") {
            return baseURL + randomLink
        }
        return randomLink
    }
    
    return ""
}

func extractBaseURL(url string) string {
    // Simple extraction - would need more robust parsing in production
    parts := strings.Split(url, "/")
    if len(parts) >= 3 {
        return parts[0] + "//" + parts[2]
    }
    return ""
}
