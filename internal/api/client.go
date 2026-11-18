package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/charlintosh/lazyado/internal/config"
	"github.com/charlintosh/lazyado/internal/debug"
)

var clientLogger = debug.Scope("client")

const apiVersion = "7.1"

// Client is the Azure DevOps API client
type Client struct {
	httpClient   *http.Client
	gate         *gate
	ctx          context.Context
	baseURL      string
	teamURL      string
	releaseURL   string
	webURL       string
	authHeader   string
	organization string
	project      string
	team         string
	maxFileBytes int64
	maxWorkItems int
}

// NewClient creates a new Azure DevOps API client
func NewClient(cfg *config.Config) *Client {
	// Azure DevOps uses Basic auth with empty username and PAT as password
	auth := base64.StdEncoding.EncodeToString([]byte(":" + cfg.PAT))

	return &Client{
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: newTransport(),
		},
		gate:         newGate(),
		baseURL:      cfg.BaseURL(),
		teamURL:      cfg.TeamURL(),
		releaseURL:   cfg.ReleaseURL(),
		webURL:       cfg.WebURL(),
		authHeader:   "Basic " + auth,
		organization: cfg.Organization,
		project:      cfg.Project,
		team:         cfg.Team,
		maxFileBytes: cfg.DiffLimits().MaxFileBytes(),
		maxWorkItems: cfg.MaxWorkItems(),
	}
}

func newTransport() http.RoundTripper {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConnsPerHost = maxInFlight
	return t
}

func (c *Client) WithContext(ctx context.Context) *Client {
	cp := *c
	cp.ctx = ctx
	return &cp
}

func (c *Client) Throttle() Throttle {
	return c.gate.throttle()
}

func (c *Client) MaxWorkItems() int {
	return c.maxWorkItems
}

func (c *Client) Usage() Usage {
	return c.gate.totals()
}

func (c *Client) context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, url string, body io.Reader) (*http.Response, error) {
	return c.doRequestWithContentType(method, url, body, "application/json")
}

// doRequestWithContentType performs an HTTP request with authentication and custom content type
func (c *Client) doRequestWithContentType(method, url string, body io.Reader, contentType string) (*http.Response, error) {
	ctx := c.context()
	for attempt := 0; ; attempt++ {
		resp, rate, err := c.send(ctx, method, url, body, contentType)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}
		msg, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if retryable(method, resp.StatusCode) && attempt < maxRetries {
			wait := retryWait(rate, attempt)
			clientLogger.Debugf("API %d from %s, retrying in %v", resp.StatusCode, debug.RedactURL(url), wait)
			if rate.retryAfter == 0 {
				if err := c.gate.sleep(ctx, wait); err != nil {
					return nil, fmt.Errorf("executing request: %w", err)
				}
			}
			continue
		}
		clientLogger.Debugf("API error %d from %s", resp.StatusCode, debug.RedactURL(url))
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("Azure DevOps is rate limiting requests, try again in a minute: %s", extractAPIMessage(msg))
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, extractAPIMessage(msg))
	}
}

func (c *Client) send(ctx context.Context, method, url string, body io.Reader, contentType string) (*http.Response, rateHeaders, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, rateHeaders{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", contentType)

	if err := c.gate.acquire(ctx); err != nil {
		return nil, rateHeaders{}, fmt.Errorf("executing request: %w", err)
	}
	defer c.gate.release()

	clientLogger.Debugf("fetching %s %s", method, debug.RedactURL(url))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		clientLogger.Debugf("error fetching %s", debug.RedactURL(url))
		return nil, rateHeaders{}, fmt.Errorf("executing request: %w", err)
	}
	rate := c.gate.observe(resp.StatusCode, resp.Header)
	usage := c.gate.totals()
	clientLogger.Debugf("fetched %s %d cost=%.4f session=%d/%.3f", debug.RedactURL(url), resp.StatusCode, rate.cost, usage.Requests, usage.Cost)
	return resp, rate, nil
}

func versioned(baseURL, endpoint, version string) string {
	url := baseURL + endpoint
	if !strings.HasPrefix(endpoint, "/") {
		url = baseURL + "/" + endpoint
	}
	separator := "?"
	if strings.Contains(url, "?") {
		separator = "&"
	}
	return url + separator + "api-version=" + version
}

// get performs a GET request to base URL
func (c *Client) get(endpoint string) (*http.Response, error) {
	return c.doRequest(http.MethodGet, versioned(c.baseURL, endpoint, apiVersion), nil)
}

// getTeam performs a GET request to team-specific URL
func (c *Client) getTeam(endpoint string) (*http.Response, error) {
	return c.doRequest(http.MethodGet, versioned(c.teamURL, endpoint, apiVersion), nil)
}

// getPreview performs a GET request with preview API version
func (c *Client) getPreview(endpoint string) (*http.Response, error) {
	return c.doRequest(http.MethodGet, versioned(c.baseURL, endpoint, apiVersion+"-preview"), nil)
}

// post performs a POST request
func (c *Client) post(endpoint string, body io.Reader) (*http.Response, error) {
	return c.doRequest(http.MethodPost, versioned(c.baseURL, endpoint, apiVersion), body)
}

// postPreview performs a POST request with preview API version
func (c *Client) postPreview(endpoint string, body io.Reader) (*http.Response, error) {
	return c.doRequest(http.MethodPost, versioned(c.baseURL, endpoint, apiVersion+"-preview"), body)
}

// patch performs a PATCH request (for work item updates)
func (c *Client) patch(endpoint string, body io.Reader) (*http.Response, error) {
	return c.doRequestWithContentType(http.MethodPatch, versioned(c.baseURL, endpoint, apiVersion), body, "application/json-patch+json")
}

// patchPreview performs a PATCH request with preview API version
func (c *Client) patchPreview(endpoint string, body io.Reader) (*http.Response, error) {
	return c.doRequest(http.MethodPatch, versioned(c.baseURL, endpoint, apiVersion+"-preview"), body)
}

// deletePreview performs a DELETE request with preview API version
func (c *Client) deletePreview(endpoint string) (*http.Response, error) {
	return c.doRequest(http.MethodDelete, versioned(c.baseURL, endpoint, apiVersion+"-preview"), nil)
}

func (c *Client) delete(endpoint string) (*http.Response, error) {
	return c.doRequest(http.MethodDelete, versioned(c.baseURL, endpoint, apiVersion), nil)
}

// decode decodes a JSON response into the given target
func decode(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// WorkItemWebURL returns the web URL for a work item
func (c *Client) WorkItemWebURL(id int) string {
	return fmt.Sprintf("%s/_workitems/edit/%d", c.webURL, id)
}

// Organization returns the organization name
func (c *Client) Organization() string {
	return c.organization
}

// Project returns the project name
func (c *Client) Project() string {
	return c.project
}

// Team returns the team name
func (c *Client) Team() string {
	return c.team
}

func extractAPIMessage(body []byte) string {
	var apiErr struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		return apiErr.Message
	}
	return string(body)
}
