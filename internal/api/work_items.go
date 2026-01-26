package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/charlintosh/lazyado/internal/debug"
	"github.com/charlintosh/lazyado/internal/models"
)

// wiqlRequest represents a WIQL query request
type wiqlRequest struct {
	Query string `json:"query"`
}

// wiqlResponse represents the response from a WIQL query
type wiqlResponse struct {
	WorkItems []struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	} `json:"workItems"`
}

// workItemsResponse represents the response for batch work item fetch
type workItemsResponse struct {
	Count int               `json:"count"`
	Value []workItemAPIItem `json:"value"`
}

// workItemAPIItem represents a work item from the API
type workItemAPIItem struct {
	ID     int            `json:"id"`
	Rev    int            `json:"rev"`
	Fields workItemFields `json:"fields"`
	URL    string         `json:"url"`
}

type workItemFields struct {
	ID           int    `json:"System.Id"`
	Title        string `json:"System.Title"`
	State        string `json:"System.State"`
	WorkItemType string `json:"System.WorkItemType"`
	AssignedTo   *struct {
		DisplayName string `json:"displayName"`
		UniqueName  string `json:"uniqueName"`
	} `json:"System.AssignedTo"`
	IterationPath      string    `json:"System.IterationPath"`
	AreaPath           string    `json:"System.AreaPath"`
	Description        string    `json:"System.Description"`
	AcceptanceCriteria string    `json:"Microsoft.VSTS.Common.AcceptanceCriteria"`
	Tags               string    `json:"System.Tags"`
	Parent             int       `json:"System.Parent"`
	Priority           int       `json:"Microsoft.VSTS.Common.Priority"`
	Effort             float64   `json:"Microsoft.VSTS.Scheduling.Effort"`
	CreatedDate        time.Time `json:"System.CreatedDate"`
	ChangedDate        time.Time `json:"System.ChangedDate"`
}

// escapeWIQL escapes a string value for use in WIQL queries
func escapeWIQL(s string) string {
	// Escape single quotes by doubling them
	return strings.ReplaceAll(s, "'", "''")
}

// QueryWorkItems queries work items using WIQL
func (c *Client) QueryWorkItems(sprintPath, state, assigned, areaPath string) ([]models.WorkItem, error) {
	// Build WIQL query
	query := `SELECT [System.Id], [System.Title], [System.State], [System.WorkItemType]
FROM WorkItems
WHERE [System.TeamProject] = @project`

	// Add sprint filter
	if sprintPath != "" && sprintPath != "all" {
		query += fmt.Sprintf(`
  AND [System.IterationPath] = '%s'`, escapeWIQL(sprintPath))
	}

	// Add state filter
	if state != "" && state != "all" {
		query += fmt.Sprintf(`
  AND [System.State] = '%s'`, escapeWIQL(state))
	}

	// Add assigned filter
	if assigned == "me" {
		query += `
  AND [System.AssignedTo] = @me`
	}

	// Add area filter
	if areaPath != "" && areaPath != "all" {
		// Clean up the path
		areaPath = strings.TrimPrefix(areaPath, "\\")
		areaPath = strings.TrimSuffix(areaPath, "\\")
		query += fmt.Sprintf(`
  AND [System.AreaPath] UNDER '%s'`, escapeWIQL(areaPath))
	}

	query += `
ORDER BY [System.ChangedDate] DESC`

	// Execute WIQL query
	reqBody := wiqlRequest{Query: query}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling WIQL request: %w", err)
	}

	resp, err := c.post("/wit/wiql", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var wiqlResp wiqlResponse
	if err := decode(resp, &wiqlResp); err != nil {
		return nil, err
	}

	if len(wiqlResp.WorkItems) == 0 {
		return []models.WorkItem{}, nil
	}

	// Get the IDs
	ids := make([]string, 0, len(wiqlResp.WorkItems))
	for _, wi := range wiqlResp.WorkItems {
		ids = append(ids, fmt.Sprintf("%d", wi.ID))
	}

	// Fetch the full work items
	return c.GetWorkItems(ids)
}

// GetWorkItems fetches multiple work items by ID
func (c *Client) GetWorkItems(ids []string) ([]models.WorkItem, error) {
	if len(ids) == 0 {
		return []models.WorkItem{}, nil
	}

	// API has a limit of 200 items per request
	const batchSize = 200
	var allItems []models.WorkItem

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		batch := ids[i:end]
		fields := "System.Id,System.Title,System.State,System.WorkItemType,System.AssignedTo,System.IterationPath,System.AreaPath,System.Description,Microsoft.VSTS.Common.AcceptanceCriteria,System.Tags,System.Parent,Microsoft.VSTS.Common.Priority,Microsoft.VSTS.Scheduling.Effort,System.CreatedDate,System.ChangedDate"

		endpoint := fmt.Sprintf("/wit/workitems?ids=%s&fields=%s", strings.Join(batch, ","), fields)
		resp, err := c.get(endpoint)
		if err != nil {
			return nil, err
		}

		var apiResp workItemsResponse
		if err := decode(resp, &apiResp); err != nil {
			return nil, err
		}

		for _, item := range apiResp.Value {
			wi := c.convertWorkItem(item)
			allItems = append(allItems, wi)
		}
	}

	// Fetch parent titles
	c.populateParentTitles(allItems)

	// Organize into hierarchy
	c.organizeHierarchy(allItems)

	return allItems, nil
}

// populateParentTitles fetches titles for all parent work items
func (c *Client) populateParentTitles(items []models.WorkItem) {
	// Collect unique parent IDs
	parentIDs := make(map[int]bool)
	for _, item := range items {
		if item.ParentID > 0 {
			parentIDs[item.ParentID] = true
		}
	}

	if len(parentIDs) == 0 {
		return
	}

	// Convert to string slice
	ids := make([]string, 0, len(parentIDs))
	for id := range parentIDs {
		ids = append(ids, fmt.Sprintf("%d", id))
	}

	// Fetch parent work items (only need ID and Title)
	endpoint := fmt.Sprintf("/wit/workitems?ids=%s&fields=System.Id,System.Title", strings.Join(ids, ","))
	resp, err := c.get(endpoint)
	if err != nil {
		return // Silently fail - parent titles are optional
	}

	var apiResp workItemsResponse
	if err := decode(resp, &apiResp); err != nil {
		return
	}

	// Build ID -> Title map
	titleMap := make(map[int]string)
	for _, item := range apiResp.Value {
		titleMap[item.ID] = item.Fields.Title
	}

	// Update items with parent titles
	for i := range items {
		if items[i].ParentID > 0 {
			if title, ok := titleMap[items[i].ParentID]; ok {
				items[i].ParentTitle = title
			}
		}
	}
}

// organizeHierarchy builds parent-child relationships in the work items
func (c *Client) organizeHierarchy(items []models.WorkItem) {
	// Build ID -> index map
	itemMap := make(map[int]*models.WorkItem)
	for i := range items {
		itemMap[items[i].ID] = &items[i]
	}

	// Link children to parents
	for i := range items {
		if items[i].ParentID > 0 {
			if parent, ok := itemMap[items[i].ParentID]; ok {
				parent.Children = append(parent.Children, &items[i])
			}
		}
	}
}

// GetWorkItem fetches a single work item by ID
func (c *Client) GetWorkItem(id int) (*models.WorkItem, error) {
	fields := "System.Id,System.Title,System.State,System.WorkItemType,System.AssignedTo,System.IterationPath,System.AreaPath,System.Description,System.Tags,System.Parent,Microsoft.VSTS.Common.Priority,System.CreatedDate,System.ChangedDate"

	endpoint := fmt.Sprintf("/wit/workitems/%d?fields=%s", id, fields)
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, err
	}

	var item workItemAPIItem
	if err := decode(resp, &item); err != nil {
		return nil, err
	}

	wi := c.convertWorkItem(item)

	// Fetch parent title if parent exists
	if wi.ParentID > 0 {
		parentEndpoint := fmt.Sprintf("/wit/workitems/%d?fields=System.Title", wi.ParentID)
		parentResp, err := c.get(parentEndpoint)
		if err == nil {
			var parentItem workItemAPIItem
			if decode(parentResp, &parentItem) == nil {
				wi.ParentTitle = parentItem.Fields.Title
			}
		}
	}

	return &wi, nil
}

// convertWorkItem converts an API work item to our model
func (c *Client) convertWorkItem(item workItemAPIItem) models.WorkItem {
	wi := models.WorkItem{
		ID:                 item.ID,
		Rev:                item.Rev,
		Title:              item.Fields.Title,
		State:              models.WorkItemState(item.Fields.State),
		Type:               models.WorkItemType(item.Fields.WorkItemType),
		IterationPath:      item.Fields.IterationPath,
		AreaPath:           item.Fields.AreaPath,
		Description:        stripHTML(item.Fields.Description),
		AcceptanceCriteria: stripHTML(item.Fields.AcceptanceCriteria),
		ParentID:           item.Fields.Parent,
		Priority:           item.Fields.Priority,
		Effort:             item.Fields.Effort,
		CreatedDate:        item.Fields.CreatedDate,
		ChangedDate:        item.Fields.ChangedDate,
		URL:                item.URL,
		WebURL:             c.WorkItemWebURL(item.ID),
	}

	if item.Fields.AssignedTo != nil {
		wi.AssignedTo = item.Fields.AssignedTo.DisplayName
	}

	// Parse tags
	if item.Fields.Tags != "" {
		tags := strings.Split(item.Fields.Tags, ";")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				wi.Tags = append(wi.Tags, tag)
			}
		}
	}

	return wi
}

// UpdateWorkItemState updates a work item's state
func (c *Client) UpdateWorkItemState(id int, newState string) error {
	// Azure DevOps uses JSON Patch format
	patchDoc := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/fields/System.State",
			"value": newState,
		},
	}

	bodyBytes, err := json.Marshal(patchDoc)
	if err != nil {
		return fmt.Errorf("marshaling patch document: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workitems/%d", id)
	resp, err := c.patch(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// AssignWorkItem assigns a work item to a user
// Pass empty string to unassign
func (c *Client) AssignWorkItem(id int, userEmail string) error {
	// Azure DevOps uses JSON Patch format
	patchDoc := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/fields/System.AssignedTo",
			"value": userEmail,
		},
	}

	bodyBytes, err := json.Marshal(patchDoc)
	if err != nil {
		return fmt.Errorf("marshaling patch document: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workitems/%d", id)
	resp, err := c.patch(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// CreateWorkItem creates a new work item
func (c *Client) CreateWorkItem(workItemType, title, description string, parentID int, areaPath, iterationPath, assignedTo string) (int, error) {
	// Build the patch document for creation
	patchDoc := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/fields/System.Title",
			"value": title,
		},
		{
			"op":    "add",
			"path":  "/fields/System.WorkItemType",
			"value": workItemType,
		},
	}

	// Add optional fields
	if description != "" {
		// Azure DevOps uses HTML format for description
		// Escape HTML and wrap with line breaks
		escapedDesc := html.EscapeString(description)
		// Replace newlines with <br/> tags
		escapedDesc = strings.ReplaceAll(escapedDesc, "\n", "<br/>")
		htmlDescription := fmt.Sprintf("<div>%s</div>", escapedDesc)
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.Description",
			"value": htmlDescription,
		})
	}

	if areaPath != "" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.AreaPath",
			"value": areaPath,
		})
	}

	if iterationPath != "" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.IterationPath",
			"value": iterationPath,
		})
	}

	if assignedTo != "" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.AssignedTo",
			"value": assignedTo,
		})
	}

	bodyBytes, err := json.Marshal(patchDoc)
	if err != nil {
		return 0, fmt.Errorf("marshaling patch document: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workitems/$%s", workItemType)
	resp, err := c.patch(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result workItemAPIItem
	if err := decode(resp, &result); err != nil {
		return 0, err
	}

	taskID := result.ID

	// If there's a parent, add the parent-child relationship using relations API
	if parentID > 0 {
		err = c.addParentRelation(taskID, parentID)
		if err != nil {
			// Don't fail the whole operation if relation fails, but log it
			// The work item was created successfully
			return taskID, fmt.Errorf("work item created but failed to add parent relation: %w", err)
		}
	}

	return taskID, nil
}

// CreateParentWorkItem creates a parent work item (PBI or Bug) with state and effort
func (c *Client) CreateParentWorkItem(workItemType, title, description, acceptanceCriteria, state, iterationPath, areaPath, assignedTo string, effort float64, priority int, tags []string, defectCause string) (int, error) {
	// Build the JSON Patch document
	patchDoc := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/fields/System.Title",
			"value": title,
		},
		{
			"op":    "add",
			"path":  "/fields/System.WorkItemType",
			"value": workItemType,
		},
	}

	// Add state if provided
	if state != "" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.State",
			"value": state,
		})
	}

	// Add description if provided
	if description != "" {
		escapedDesc := html.EscapeString(description)
		escapedDesc = strings.ReplaceAll(escapedDesc, "\n", "<br/>")
		htmlDescription := fmt.Sprintf("<div>%s</div>", escapedDesc)
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.Description",
			"value": htmlDescription,
		})
	}

	// Add acceptance criteria if provided
	if acceptanceCriteria != "" {
		escapedAC := html.EscapeString(acceptanceCriteria)
		escapedAC = strings.ReplaceAll(escapedAC, "\n", "<br/>")
		htmlAC := fmt.Sprintf("<div>%s</div>", escapedAC)
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/Microsoft.VSTS.Common.AcceptanceCriteria",
			"value": htmlAC,
		})
	}

	// Add area path if provided
	if areaPath != "" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.AreaPath",
			"value": areaPath,
		})
	}

	// Add iteration path if provided
	if iterationPath != "" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.IterationPath",
			"value": iterationPath,
		})
	}

	// Add assignee if provided
	if assignedTo != "" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.AssignedTo",
			"value": assignedTo,
		})
	}

	// Add effort if > 0 (only for PBI)
	if effort > 0 && workItemType == "Product Backlog Item" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/Microsoft.VSTS.Scheduling.Effort",
			"value": effort,
		})
	}

	// Add priority if > 0 (for both PBI and Bug)
	if priority > 0 {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/Microsoft.VSTS.Common.Priority",
			"value": priority,
		})
	}

	// Add tags if provided
	if len(tags) > 0 {
		tagsString := strings.Join(tags, "; ")
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.Tags",
			"value": tagsString,
		})
	}

	// Add defect cause if provided (only for Bug)
	if defectCause != "" && workItemType == "Bug" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/Microsoft.VSTS.CMMI.FoundInEnvironment",
			"value": defectCause,
		})
	}

	bodyBytes, err := json.Marshal(patchDoc)
	if err != nil {
		return 0, fmt.Errorf("marshaling patch document: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workitems/$%s", workItemType)
	resp, err := c.patch(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result workItemAPIItem
	if err := decode(resp, &result); err != nil {
		return 0, err
	}

	return result.ID, nil
}

// addParentRelation adds a parent-child relationship between work items
func (c *Client) addParentRelation(childID, parentID int) error {
	patchDoc := []map[string]interface{}{
		{
			"op":   "add",
			"path": "/relations/-",
			"value": map[string]interface{}{
				"rel": "System.LinkTypes.Hierarchy-Reverse",
				"url": fmt.Sprintf("%s/wit/workItems/%d", c.baseURL, parentID),
			},
		},
	}

	bodyBytes, err := json.Marshal(patchDoc)
	if err != nil {
		return fmt.Errorf("marshaling patch document: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workitems/%d", childID)
	resp, err := c.patch(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// UpdateWorkItemTitle updates a work item's title
func (c *Client) UpdateWorkItemTitle(id int, newTitle string) error {
	return c.UpdateWorkItem(id, newTitle, "")
}

// UpdateWorkItem updates a work item's title and/or description
func (c *Client) UpdateWorkItem(id int, newTitle, description string) error {
	patchDoc := []map[string]interface{}{}

	if newTitle != "" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.Title",
			"value": newTitle,
		})
	}

	if description != "" {
		// Azure DevOps uses HTML format for description
		escapedDesc := html.EscapeString(description)
		escapedDesc = strings.ReplaceAll(escapedDesc, "\n", "<br/>")
		htmlDescription := fmt.Sprintf("<div>%s</div>", escapedDesc)
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.Description",
			"value": htmlDescription,
		})
	}

	if len(patchDoc) == 0 {
		return nil
	}

	bodyBytes, err := json.Marshal(patchDoc)
	if err != nil {
		return fmt.Errorf("marshaling patch document: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workitems/%d", id)
	resp, err := c.patch(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// UpdatePBI updates a PBI's title, description, priority, effort, tags, and assigned user
func (c *Client) UpdatePBI(id int, title, description string, priority int, effort float64, tags []string, assignedTo string) error {
	patchDoc := []map[string]interface{}{}

	if title != "" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.Title",
			"value": title,
		})
	}

	if description != "" {
		// Azure DevOps uses HTML format for description
		escapedDesc := html.EscapeString(description)
		escapedDesc = strings.ReplaceAll(escapedDesc, "\n", "<br/>")
		htmlDescription := fmt.Sprintf("<div>%s</div>", escapedDesc)
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.Description",
			"value": htmlDescription,
		})
	}

	if priority > 0 {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/Microsoft.VSTS.Common.Priority",
			"value": priority,
		})
	}

	if effort > 0 {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/Microsoft.VSTS.Scheduling.Effort",
			"value": effort,
		})
	}

	if len(tags) > 0 {
		// Tags are stored as semicolon-separated string
		tagsStr := strings.Join(tags, "; ")
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.Tags",
			"value": tagsStr,
		})
	}

	if assignedTo != "" {
		patchDoc = append(patchDoc, map[string]interface{}{
			"op":    "add",
			"path":  "/fields/System.AssignedTo",
			"value": assignedTo,
		})
	}

	if len(patchDoc) == 0 {
		return nil
	}

	bodyBytes, err := json.Marshal(patchDoc)
	if err != nil {
		return fmt.Errorf("marshaling patch document: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workitems/%d", id)
	resp, err := c.patch(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// DeleteWorkItem deletes a work item
func (c *Client) DeleteWorkItem(id int) error {
	endpoint := fmt.Sprintf("/wit/workitems/%d", id)
	url := fmt.Sprintf("%s%s?api-version=%s", c.baseURL, endpoint, apiVersion)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

var commentsLogger = debug.Scope("api.comments")

// GetComments fetches comments for a work item
func (c *Client) GetComments(workItemID int) ([]models.Comment, error) {
	// Request expanded data including mentions and rendered text
	endpoint := fmt.Sprintf("/wit/workItems/%d/comments?$expand=all", workItemID)
	resp, err := c.getPreview(endpoint)
	if err != nil {
		return nil, err
	}

	// DEBUG: Log the raw API response body for troubleshooting
	if resp != nil && resp.Body != nil {
		var rawBody bytes.Buffer
		tee := io.TeeReader(resp.Body, &rawBody)
		// Try to decode as JSON for pretty logging
		var pretty map[string]interface{}
		if err := json.NewDecoder(tee).Decode(&pretty); err == nil {
			prettyJSON, _ := json.MarshalIndent(pretty, "", "  ")
			commentsLogger.Debugf("Raw API comment response (pretty):\n%s", string(prettyJSON))
		} else {
			// Fallback: log as plain text
			all, _ := io.ReadAll(&rawBody)
			commentsLogger.Debugf("Raw API comment response (raw):\n%s", string(all))
		}
		// Reset resp.Body for downstream decode
		resp.Body = io.NopCloser(&rawBody)
	}

	// Response body will be decoded directly; do not log full response on success.

	// Azure DevOps uses "comments" field (not "value") for work item comments
	var apiResp struct {
		Count      int `json:"count"`
		TotalCount int `json:"totalCount"`
		Comments   []struct {
			ID           int    `json:"id"`
			Text         string `json:"text"`
			RenderedText string `json:"renderedText,omitempty"`
			Format       string `json:"format,omitempty"`
			WorkItemID   int    `json:"workItemId"`
			Version      int    `json:"version"`
			Mentions     []struct {
				ArtifactID   string `json:"artifactId"`
				ArtifactType string `json:"artifactType"`
				TargetID     string `json:"targetId"`
			} `json:"mentions,omitempty"`
			CreatedBy struct {
				DisplayName string `json:"displayName"`
			} `json:"createdBy"`
			CreatedDate time.Time `json:"createdDate"`
			ModifiedBy  struct {
				DisplayName string `json:"displayName"`
			} `json:"modifiedBy,omitempty"`
			ModifiedDate time.Time `json:"modifiedDate,omitempty"`
		} `json:"comments"`
	}

	if err := decode(resp, &apiResp); err != nil {
		return nil, err
	}
	// Do not log the full response on success; centralized logging in client.doRequest

	comments := make([]models.Comment, 0, len(apiResp.Comments))
	for _, c := range apiResp.Comments {
		// Map mentions
		mentions := make([]models.CommentMention, 0, len(c.Mentions))
		for _, m := range c.Mentions {
			mentions = append(mentions, models.CommentMention{
				ArtifactID:   m.ArtifactID,
				ArtifactType: m.ArtifactType,
				TargetID:     m.TargetID,
			})
		}

		comment := models.Comment{
			ID:           c.ID,
			Text:         stripHTML(html.UnescapeString(c.Text)),
			RenderedText: html.UnescapeString(c.RenderedText),
			Format:       c.Format,
			Mentions:     mentions,
			CreatedBy:    c.CreatedBy.DisplayName,
			CreatedDate:  c.CreatedDate,
			ModifiedBy:   c.ModifiedBy.DisplayName,
			ModifiedDate: c.ModifiedDate,
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

// CreateComment creates a new comment on a work item
func (c *Client) CreateComment(workItemID int, text string) error {
	payload := map[string]interface{}{
		"text":   text,
		"format": "html", // Specify that we're sending HTML content
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling comment payload: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workItems/%d/comments", workItemID)
	resp, err := c.postPreview(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateComment updates an existing comment on a work item
func (c *Client) UpdateComment(workItemID, commentID int, text string) error {
	payload := map[string]interface{}{
		"text": text,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling comment payload: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workItems/%d/comments/%d", workItemID, commentID)
	resp, err := c.patchPreview(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("updating comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteComment deletes a comment from a work item
func (c *Client) DeleteComment(workItemID, commentID int) error {
	endpoint := fmt.Sprintf("/wit/workItems/%d/comments/%d", workItemID, commentID)
	resp, err := c.deletePreview(endpoint)
	if err != nil {
		return fmt.Errorf("deleting comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// stripHTML removes HTML tags from a string
func stripHTML(s string) string {
	// Simple HTML tag removal
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")

	// Replace common HTML entities
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")

	// Trim whitespace
	s = strings.TrimSpace(s)

	return s
}
