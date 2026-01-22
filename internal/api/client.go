package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/charlintosh/lazyado/internal/config"
	"github.com/charlintosh/lazyado/internal/debug"
)

var clientLogger = debug.Scope("client")

const apiVersion = "7.1"

// Client is the Azure DevOps API client
type Client struct {
	httpClient   *http.Client
	baseURL      string
	teamURL      string
	webURL       string
	authHeader   string
	organization string
	project      string
	team         string
}

// NewClient creates a new Azure DevOps API client
func NewClient(cfg *config.Config) *Client {
	// Azure DevOps uses Basic auth with empty username and PAT as password
	auth := base64.StdEncoding.EncodeToString([]byte(":" + cfg.PAT))

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:      cfg.BaseURL(),
		teamURL:      cfg.TeamURL(),
		webURL:       cfg.WebURL(),
		authHeader:   "Basic " + auth,
		organization: cfg.Organization,
		project:      cfg.Project,
		team:         cfg.Team,
	}
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, url string, body io.Reader) (*http.Response, error) {
	return c.doRequestWithContentType(method, url, body, "application/json")
}

// doRequestWithContentType performs an HTTP request with authentication and custom content type
func (c *Client) doRequestWithContentType(method, url string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", contentType)

	// Log start
	clientLogger.Debugf("fetching %s %s", method, url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		clientLogger.Debugf("error fetching %s %v", url, err)
		return nil, fmt.Errorf("executing request: %w", err)
	}

	// Log completion
	clientLogger.Debugf("fetched %s %d", url, resp.StatusCode)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		// Log error body for debugging
		clientLogger.Debugf("API error %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// get performs a GET request to base URL
func (c *Client) get(endpoint string) (*http.Response, error) {
	return c.getWithBase(c.baseURL, endpoint)
}

// getTeam performs a GET request to team-specific URL
func (c *Client) getTeam(endpoint string) (*http.Response, error) {
	return c.getWithBase(c.teamURL, endpoint)
}

// getPreview performs a GET request with preview API version
func (c *Client) getPreview(endpoint string) (*http.Response, error) {
	return c.getWithVersion(c.baseURL, endpoint, apiVersion+"-preview")
}

// getWithBase performs a GET request with a specific base URL
func (c *Client) getWithBase(baseURL, endpoint string) (*http.Response, error) {
	return c.getWithVersion(baseURL, endpoint, apiVersion)
}

// getWithVersion performs a GET request with a specific API version
func (c *Client) getWithVersion(baseURL, endpoint, version string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", baseURL, endpoint)
	if endpoint[0] != '/' {
		url = fmt.Sprintf("%s/%s", baseURL, endpoint)
	}

	// Add API version
	if len(url) > 0 {
		separator := "?"
		if len(url) > 0 && url[len(url)-1] != '?' {
			for _, c := range url {
				if c == '?' {
					separator = "&"
					break
				}
			}
		}
		url = fmt.Sprintf("%s%sapi-version=%s", url, separator, version)
	}

	return c.doRequest("GET", url, nil)
}

// post performs a POST request
func (c *Client) post(endpoint string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	if endpoint[0] != '/' {
		url = fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	}

	// Add API version
	separator := "?"
	for _, ch := range url {
		if ch == '?' {
			separator = "&"
			break
		}
	}
	url = fmt.Sprintf("%s%sapi-version=%s", url, separator, apiVersion)

	return c.doRequest("POST", url, body)
}

// postPreview performs a POST request with preview API version
func (c *Client) postPreview(endpoint string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	if endpoint[0] != '/' {
		url = fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	}

	// Add API version with preview
	separator := "?"
	for _, ch := range url {
		if ch == '?' {
			separator = "&"
			break
		}
	}
	url = fmt.Sprintf("%s%sapi-version=%s-preview", url, separator, apiVersion)

	return c.doRequest("POST", url, body)
}

// patch performs a PATCH request (for work item updates)
func (c *Client) patch(endpoint string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	if endpoint[0] != '/' {
		url = fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	}

	// Add API version
	separator := "?"
	for _, ch := range url {
		if ch == '?' {
			separator = "&"
			break
		}
	}
	url = fmt.Sprintf("%s%sapi-version=%s", url, separator, apiVersion)

	return c.doRequestWithContentType("PATCH", url, body, "application/json-patch+json")
}

// patchPreview performs a PATCH request with preview API version
func (c *Client) patchPreview(endpoint string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	if endpoint[0] != '/' {
		url = fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	}

	// Add API version with preview
	separator := "?"
	for _, ch := range url {
		if ch == '?' {
			separator = "&"
			break
		}
	}
	url = fmt.Sprintf("%s%sapi-version=%s-preview", url, separator, apiVersion)

	return c.doRequestWithContentType("PATCH", url, body, "application/json")
}

// deletePreview performs a DELETE request with preview API version
func (c *Client) deletePreview(endpoint string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	if endpoint[0] != '/' {
		url = fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	}

	// Add API version with preview
	separator := "?"
	for _, ch := range url {
		if ch == '?' {
			separator = "&"
			break
		}
	}
	url = fmt.Sprintf("%s%sapi-version=%s-preview", url, separator, apiVersion)

	return c.doRequest("DELETE", url, nil)
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
