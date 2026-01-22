package api

import (
	"sort"
	"strings"

	"github.com/charlintosh/lazyado/internal/models"
)

// classificationNode represents an area node from the API
type classificationNode struct {
	ID          int                  `json:"id"`
	Identifier  string               `json:"identifier"`
	Name        string               `json:"name"`
	Path        string               `json:"path"`
	HasChildren bool                 `json:"hasChildren"`
	Children    []classificationNode `json:"children"`
}

// GetAreas fetches all areas for the project
func (c *Client) GetAreas() ([]models.Area, error) {
	// Use the classification nodes API with depth to get area hierarchy
	resp, err := c.get("/wit/classificationnodes/areas?$depth=10")
	if err != nil {
		return nil, err
	}

	var rootNode classificationNode
	if err := decode(resp, &rootNode); err != nil {
		return nil, err
	}

	// Flatten the tree into a list
	areas := flattenAreas(rootNode, "")

	// Sort areas by path for consistent ordering
	sort.Slice(areas, func(i, j int) bool {
		return areas[i].Path < areas[j].Path
	})

	return areas, nil
}

// flattenAreas recursively flattens the area tree
func flattenAreas(node classificationNode, parentPath string) []models.Area {
	var areas []models.Area

	// Build the path
	path := node.Path
	if path == "" {
		path = node.Name
	}

	// Clean up the path:
	// 1. Remove leading backslash
	// 2. Remove "\Area" from path (API returns \Project\Area\Team but work items use Project\Team)
	path = strings.TrimPrefix(path, "\\")
	path = strings.TrimSuffix(path, "\\")

	// Remove the "Area" segment from the path (e.g., "Project\Area\Team" -> "Project\Team")
	parts := strings.Split(path, "\\")
	if len(parts) >= 2 && parts[1] == "Area" {
		// Remove the "Area" part
		newParts := []string{parts[0]}
		if len(parts) > 2 {
			newParts = append(newParts, parts[2:]...)
		}
		path = strings.Join(newParts, "\\")
	}

	// Add this node
	areas = append(areas, models.Area{
		ID:   node.ID,
		Name: node.Name,
		Path: path,
	})

	// Recursively add children
	for _, child := range node.Children {
		childAreas := flattenAreas(child, path)
		areas = append(areas, childAreas...)
	}

	return areas
}

// GetAreaDisplayName returns a shortened display name for an area
func GetAreaDisplayName(path string) string {
	parts := strings.Split(path, "\\")
	if len(parts) > 1 {
		// Return the last part for brevity
		return parts[len(parts)-1]
	}
	return path
}
