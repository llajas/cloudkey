package network

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"
)

// UDMProClient represents a UniFi controller client
type UDMProClient struct {
	BaseURL    string
	Username   string
	Password   string
	Site       string
	Version    string
	HTTPClient *http.Client
	IsUniFiOS  bool
	AuthToken  string
	CSRFToken  string
	cache      *SpeedtestCache
	session    *SessionCache
	cacheMutex sync.RWMutex
}

// SpeedtestCache represents a cached speedtest result
type SpeedtestCache struct {
	Result    *SpeedtestResult
	Timestamp time.Time
	TTL       time.Duration
}

// SessionCache represents cached authentication session
type SessionCache struct {
	AuthToken string
	CSRFToken string
	Expires   time.Time
}

// SpeedtestResult represents a single speedtest result
type SpeedtestResult struct {
	DownloadMbps float64 `json:"download_mbps"`
	UploadMbps   float64 `json:"upload_mbps"`
	LatencyMs    float64 `json:"latency_ms"`
	Timestamp    int64   `json:"timestamp"`
}

// LoginRequest represents the login payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Meta struct {
		RC  string `json:"rc"`
		Msg string `json:"msg,omitempty"`
	} `json:"meta"`
	Data []struct{} `json:"data,omitempty"`
}

// SpeedtestRequest represents the speedtest API request
type SpeedtestRequest struct {
	Attrs []string `json:"attrs"`
	Start int64    `json:"start,omitempty"`
	End   int64    `json:"end,omitempty"`
}

// SpeedtestResponse represents the speedtest API response
type SpeedtestResponse struct {
	Meta struct {
		RC  string `json:"rc"`
		Msg string `json:"msg,omitempty"`
	} `json:"meta"`
	Data []struct {
		XputDownload float64 `json:"xput_download"`
		XputUpload   float64 `json:"xput_upload"`
		Latency      float64 `json:"latency"`
		Time         int64   `json:"time"`
	} `json:"data"`
}

// NewUDMProClient creates a new UDM Pro API client
func NewUDMProClient(baseURL, username, password, site, version string) (*UDMProClient, error) {
	// Create cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}

	// Create HTTP client with TLS config (skip verification for local networks)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &UDMProClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Username:   username,
		Password:   password,
		Site:       site,
		Version:    version,
		HTTPClient: &http.Client{Transport: transport, Jar: jar, Timeout: 30 * time.Second},
		cache: &SpeedtestCache{
			TTL: 24 * time.Hour, // Cache for 24 hours since tests run daily
		},
		session: &SessionCache{
			Expires: time.Now(), // Expired initially
		},
	}

	// Detect controller type
	if err := client.detectControllerType(); err != nil {
		return nil, fmt.Errorf("failed to detect controller type: %v", err)
	}

	return client, nil
}

// detectControllerType determines if we're dealing with a UniFi OS controller
func (c *UDMProClient) detectControllerType() error {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/")
	if err != nil {
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			return fmt.Errorf("network timeout - cannot reach UDM Pro at %s. Check IP address and network connectivity", c.BaseURL)
		} else if strings.Contains(err.Error(), "connection refused") {
			return fmt.Errorf("connection refused - UDM Pro at %s is not accessible. Check if device is running and firewall settings", c.BaseURL)
		} else if strings.Contains(err.Error(), "no such host") {
			return fmt.Errorf("host not found - invalid UDM Pro address: %s. Check IP address or hostname", c.BaseURL)
		}
		return fmt.Errorf("failed to detect controller type: %v", err)
	}
	defer resp.Body.Close()

	// If we get 200, it's UniFi OS
	c.IsUniFiOS = resp.StatusCode == 200
	return nil
}

// isSessionValid checks if current session is still valid
func (c *UDMProClient) isSessionValid() bool {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	return c.session.AuthToken != "" && time.Now().Before(c.session.Expires)
}

// cacheSession stores the current session
func (c *UDMProClient) cacheSession() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.session.AuthToken = c.AuthToken
	c.session.CSRFToken = c.CSRFToken
	c.session.Expires = time.Now().Add(8 * time.Hour) // Sessions typically last 8 hours
}

// useCachedSession restores cached session
func (c *UDMProClient) useCachedSession() {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	c.AuthToken = c.session.AuthToken
	c.CSRFToken = c.session.CSRFToken
}

// Login authenticates with the UniFi controller
func (c *UDMProClient) Login() error {
	// Check if we have a valid cached session
	if c.isSessionValid() {
		fmt.Println("Using cached authentication session")
		c.useCachedSession()
		return nil
	}

	fmt.Println("No valid session - performing fresh login")

	// Determine login endpoint based on controller type
	var loginURL string
	if c.IsUniFiOS {
		loginURL = c.BaseURL + "/api/auth/login"
	} else {
		loginURL = c.BaseURL + "/api/login"
	}

	// Prepare login payload
	loginData := LoginRequest{
		Username: c.Username,
		Password: c.Password,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("failed to marshal login data: %v", err)
	}

	// Create login request (PHP client uses POST for login)
	req, err := http.NewRequest("POST", loginURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %v", err)
	}

	// Set headers matching PHP client (as single values, not arrays)
	req.Header["Accept"] = []string{"application/json"}
	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["Expect"] = []string{""}                    // Prevent 100-continue
	req.Header["Referer"] = []string{c.BaseURL + "/login"} // Match PHP client

	// Debug: Print request details (comment out for production)
	// fmt.Printf("=== Login Request Debug ===\n")
	// fmt.Printf("URL: %s\n", loginURL)
	// fmt.Printf("Method: %s\n", req.Method)
	// fmt.Printf("Headers:\n")
	// for name, values := range req.Header {
	// 	fmt.Printf("  %s: %s\n", name, values)
	// }
	// fmt.Printf("Body: %s\n", string(jsonData))
	// fmt.Printf("========================\n")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for later processing
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Debug: Print response details (comment out for production)
	// fmt.Printf("=== Login Response Debug ===\n")
	// fmt.Printf("Status Code: %d\n", resp.StatusCode)
	// fmt.Printf("Headers:\n")
	// for name, values := range resp.Header {
	// 	fmt.Printf("  %s: %s\n", name, values)
	// }
	// fmt.Printf("Response Body: %s\n", string(body))
	// fmt.Printf("========================\n")

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("login failed with status: %d (rate limited) - please wait before retrying", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	// Restore body for later processing
	resp.Body = io.NopCloser(bytes.NewReader(body))

	// Extract authentication token from cookies (matching PHP client behavior)
	for _, cookie := range resp.Cookies() {
		if c.IsUniFiOS && cookie.Name == "TOKEN" {
			c.AuthToken = cookie.Value
			// Extract CSRF token from JWT for UniFi OS
			if err := c.extractCSRFToken(); err != nil {
				return fmt.Errorf("failed to extract CSRF token: %v", err)
			}
		} else if !c.IsUniFiOS && cookie.Name == "unifises" {
			c.AuthToken = cookie.Value
		}
	}

	if c.AuthToken == "" {
		return fmt.Errorf("no authentication token found in response")
	}

	// Cache the successful session
	c.cacheSession()

	return nil
}

// extractCSRFToken extracts CSRF token from JWT token (UniFi OS only)
func (c *UDMProClient) extractCSRFToken() error {
	if !c.IsUniFiOS || c.AuthToken == "" {
		return nil
	}

	// JWT format: header.payload.signature
	parts := strings.Split(c.AuthToken, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT format - expected 3 parts, got %d", len(parts))
	}

	// Decode payload (base64url encoded)
	payload := parts[1]

	// Base64url padding fix
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	// Replace URL-safe characters with standard base64 characters
	payload = strings.ReplaceAll(payload, "-", "+")
	payload = strings.ReplaceAll(payload, "_", "/")

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	var jwtData map[string]any
	if err := json.Unmarshal(decoded, &jwtData); err != nil {
		return fmt.Errorf("failed to parse JWT payload: %v", err)
	}

	// Try different possible CSRF token field names
	possibleFields := []string{"csrfToken", "csrf_token", "xsrfToken", "token"}
	for _, field := range possibleFields {
		if token, exists := jwtData[field]; exists {
			if tokenStr, ok := token.(string); ok && tokenStr != "" {
				c.CSRFToken = tokenStr
				fmt.Printf("Extracted CSRF token from field '%s': %s...\n", field, c.CSRFToken[:min(10, len(c.CSRFToken))])
				return nil
			}
		}
	}

	// If no CSRF token found, that might be OK for some controllers
	fmt.Printf("No CSRF token found in JWT payload - this may be normal for some UniFi OS versions\n")
	return nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getCachedSpeedtest returns cached result if valid, nil otherwise
func (c *UDMProClient) getCachedSpeedtest() *SpeedtestResult {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	if c.cache.Result != nil && time.Since(c.cache.Timestamp) < c.cache.TTL {
		return c.cache.Result
	}
	return nil
}

// setCachedSpeedtest stores the speedtest result in cache
func (c *UDMProClient) setCachedSpeedtest(result *SpeedtestResult) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cache.Result = result
	c.cache.Timestamp = time.Now()
}

// GetSpeedtestResults fetches speedtest results from the controller
func (c *UDMProClient) GetSpeedtestResults() (*SpeedtestResult, error) {
	// Check cache first
	if cached := c.getCachedSpeedtest(); cached != nil {
		return cached, nil
	}

	// Default to last 24 hours
	end := time.Now().UnixMilli()
	start := end - (24 * 60 * 60 * 1000) // 24 hours ago

	result, err := c.GetSpeedtestResultsInRange(start, end)
	if err != nil {
		return nil, err
	}

	// Cache the successful result
	c.setCachedSpeedtest(result)
	return result, nil
}

// GetSpeedtestResultsInRange fetches speedtest results within a specific time range
func (c *UDMProClient) GetSpeedtestResultsInRange(start, end int64) (*SpeedtestResult, error) {
	// Build URL exactly like PHP client does
	path := fmt.Sprintf("/api/s/%s/stat/report/archive.speedtest", c.Site)
	apiURL := c.BaseURL + path

	// For UniFi OS, the PHP client automatically adds /proxy/network prefix (line 4690-4692 in PHP)
	if c.IsUniFiOS {
		apiURL = c.BaseURL + "/proxy/network" + path
	}

	speedtestReq := SpeedtestRequest{
		Attrs: []string{"xput_download", "xput_upload", "latency", "time"},
		Start: start,
		End:   end,
	}

	payload, err := json.Marshal(speedtestReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal speedtest request: %v", err)
	}

	// PHP client uses GET by default, switches to POST when payload present (line 4710-4712)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create speedtest request: %v", err)
	}

	// Since we have a payload, switch to POST (matching PHP client behavior)
	req.Method = "POST"
	req.Body = io.NopCloser(bytes.NewReader(payload))

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Expect", "")

	// Add CSRF token for UniFi OS only for POST requests (like PHP client does)
	if c.IsUniFiOS && req.Method == "POST" && c.CSRFToken != "" {
		req.Header["x-csrf-token"] = []string{c.CSRFToken}
		fmt.Printf("Adding CSRF token to speedtest request: %s\n", c.CSRFToken[:10]+"...")
	} else if c.IsUniFiOS && req.Method == "POST" {
		fmt.Printf("Warning: No CSRF token available for UniFi OS request\n")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("speedtest request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Clear expired session and retry once (matching PHP client behavior)
		c.AuthToken = ""
		c.CSRFToken = ""
		c.session.Expires = time.Now() // Mark as expired

		if err := c.Login(); err != nil {
			return nil, fmt.Errorf("re-authentication failed: %v", err)
		}
		// Retry the request with fresh authentication
		return c.GetSpeedtestResultsInRange(start, end)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("speedtest request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse response using similar logic to PHP client (lines 4373-4435)
	// Try to parse as standard UniFi API response with meta field first
	var standardResp SpeedtestResponse
	if err := json.Unmarshal(body, &standardResp); err == nil && standardResp.Meta.RC != "" {
		// Check for API error in standard format
		if standardResp.Meta.RC != "ok" {
			if standardResp.Meta.RC == "error" {
				errorMsg := "Unknown error from controller"
				if standardResp.Meta.Msg != "" {
					errorMsg = standardResp.Meta.Msg
				}
				return nil, fmt.Errorf("API error: %s", errorMsg)
			}
			return nil, fmt.Errorf("API returned status: %s", standardResp.Meta.RC)
		}
		if len(standardResp.Data) == 0 {
			return nil, fmt.Errorf("no speedtest results found in response")
		}
		// Find the most recent valid speedtest result (by timestamp, not zero values)
		var mostRecent *struct {
			XputDownload float64 `json:"xput_download"`
			XputUpload   float64 `json:"xput_upload"`
			Latency      float64 `json:"latency"`
			Time         int64   `json:"time"`
		}

		for i := range standardResp.Data {
			result := &standardResp.Data[i]
			if result.XputDownload > 0 || result.XputUpload > 0 {
				if mostRecent == nil || result.Time > mostRecent.Time {
					mostRecent = result
				}
			}
		}

		if mostRecent == nil {
			return nil, fmt.Errorf("no valid speedtest results found (all results have zero values)")
		}
		return c.convertSpeedtestResult(mostRecent), nil
	}

	// Try UniFi OS format (direct array without meta wrapper)
	var uniFiOSResults []struct {
		XputDownload float64 `json:"xput_download"`
		XputUpload   float64 `json:"xput_upload"`
		Latency      float64 `json:"latency"`
		Time         int64   `json:"time"`
	}

	if err := json.Unmarshal(body, &uniFiOSResults); err == nil && len(uniFiOSResults) > 0 {
		// Find the most recent valid speedtest result (by timestamp, not zero values)
		var mostRecent *struct {
			XputDownload float64 `json:"xput_download"`
			XputUpload   float64 `json:"xput_upload"`
			Latency      float64 `json:"latency"`
			Time         int64   `json:"time"`
		}

		for i := range uniFiOSResults {
			result := &uniFiOSResults[i]
			if result.XputDownload > 0 || result.XputUpload > 0 {
				if mostRecent == nil || result.Time > mostRecent.Time {
					mostRecent = result
				}
			}
		}

		if mostRecent == nil {
			return nil, fmt.Errorf("no valid speedtest results found (all results have zero values)")
		}
		return c.convertSpeedtestResult(mostRecent), nil
	}

	// Try v2 API format (has errorCode instead of meta)
	var v2Response struct {
		ErrorCode int    `json:"errorCode"`
		Message   string `json:"message"`
		Data      []struct {
			XputDownload float64 `json:"xput_download"`
			XputUpload   float64 `json:"xput_upload"`
			Latency      float64 `json:"latency"`
			Time         int64   `json:"time"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &v2Response); err == nil {
		if v2Response.ErrorCode != 0 {
			errorMsg := "Unknown error from v2 API"
			if v2Response.Message != "" {
				errorMsg = v2Response.Message
			}
			return nil, fmt.Errorf("v2 API error (code %d): %s", v2Response.ErrorCode, errorMsg)
		}
		if len(v2Response.Data) > 0 {
			// Find the most recent valid speedtest result (by timestamp, not zero values)
			var mostRecent *struct {
				XputDownload float64 `json:"xput_download"`
				XputUpload   float64 `json:"xput_upload"`
				Latency      float64 `json:"latency"`
				Time         int64   `json:"time"`
			}

			for i := range v2Response.Data {
				result := &v2Response.Data[i]
				if result.XputDownload > 0 || result.XputUpload > 0 {
					if mostRecent == nil || result.Time > mostRecent.Time {
						mostRecent = result
					}
				}
			}

			if mostRecent == nil {
				return nil, fmt.Errorf("no valid speedtest results found (all results have zero values)")
			}
			return c.convertSpeedtestResult(mostRecent), nil
		}
	}

	// If all parsing attempts fail, return the raw response for debugging
	return nil, fmt.Errorf("failed to parse speedtest response in any known format. Raw response: %s", string(body))
}

// convertSpeedtestResult converts API response to our format
func (c *UDMProClient) convertSpeedtestResult(data *struct {
	XputDownload float64 `json:"xput_download"`
	XputUpload   float64 `json:"xput_upload"`
	Latency      float64 `json:"latency"`
	Time         int64   `json:"time"`
}) *SpeedtestResult {
	return &SpeedtestResult{
		DownloadMbps: data.XputDownload, // API already returns Mbps
		UploadMbps:   data.XputUpload,   // API already returns Mbps
		LatencyMs:    data.Latency,
		Timestamp:    data.Time,
	}
}

// FormatSpeed formats speed values with appropriate units (Mbps/Gbps)
func FormatSpeed(mbps float64) string {
	if mbps >= 1000 {
		return fmt.Sprintf("%.1f Gb/s", mbps/1000)
	}
	return fmt.Sprintf("%.1f Mb/s", mbps)
}

// GetUDMProSpeedtest is a convenience function that creates a client and fetches results
func GetUDMProSpeedtest(baseURL, username, password, site, version string) (*SpeedtestResult, error) {
	client, err := NewUDMProClient(baseURL, username, password, site, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	if err := client.Login(); err != nil {
		return nil, fmt.Errorf("login failed: %v", err)
	}

	result, err := client.GetSpeedtestResults()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch speedtest results: %v", err)
	}

	fmt.Printf("Successfully fetched speedtest: Download=%.1f Mbps, Upload=%.1f Mbps, Latency=%.1f ms\n",
		result.DownloadMbps, result.UploadMbps, result.LatencyMs)

	return result, nil
}
