package api

import (
	"fmt"
	"strings"

	"github.com/andi/fileaction/backend/database"
	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v3"
)

// ============== Plugin Handlers ==============

// CreatePluginRequest represents the request to create a new plugin
type CreatePluginRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	YAMLContent string `json:"yaml_content"`
	CreatedBy   string `json:"created_by,omitempty"`
}

// UpdatePluginRequest represents the request to update plugin metadata
type UpdatePluginRequest struct {
	Description string `json:"description"`
	YAMLContent string `json:"yaml_content"` // When provided, creates a new version
}

// SearchPluginsRequest represents the request to search plugins
type SearchPluginsRequest struct {
	Query  string   `query:"query"`
	Source string   `query:"source"`
	Tags   []string `query:"tags"`
}

// listPlugins returns all plugins
func (s *Server) listPlugins(c *fiber.Ctx) error {
	repo := database.NewPluginRepo(s.db)
	plugins, err := repo.GetAllPlugins()
	if err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}
	return c.JSON(plugins)
}

// createPlugin creates a new plugin
func (s *Server) createPlugin(c *fiber.Ctx) error {
	var req CreatePluginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(400).JSON(ErrorResponse{Error: "Plugin name is required"})
	}
	if req.YAMLContent == "" {
		return c.Status(400).JSON(ErrorResponse{Error: "Plugin YAML content is required"})
	}

	// Validate YAML structure
	if err := validatePluginYAML(req.YAMLContent); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: fmt.Sprintf("Invalid plugin YAML: %v", err)})
	}

	repo := database.NewPluginRepo(s.db)
	plugin, version, err := repo.CreatePlugin(req.Name, req.Description, req.YAMLContent, req.CreatedBy)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "Duplicate entry") {
			return c.Status(409).JSON(ErrorResponse{Error: "Plugin with this name already exists"})
		}
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{
		"plugin":  plugin,
		"version": version,
	})
}

// getPlugin returns a plugin with all its versions
func (s *Server) getPlugin(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(ErrorResponse{Error: "Plugin ID is required"})
	}

	repo := database.NewPluginRepo(s.db)
	pluginWithVersions, err := repo.GetPluginWithVersions(id)
	if err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Plugin not found"})
	}

	return c.JSON(pluginWithVersions)
}

// updatePlugin updates plugin metadata or creates a new version
func (s *Server) updatePlugin(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(ErrorResponse{Error: "Plugin ID is required"})
	}

	var req UpdatePluginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	repo := database.NewPluginRepo(s.db)

	// If YAML content is provided, create a new version
	if req.YAMLContent != "" {
		// Validate YAML structure
		if err := validatePluginYAML(req.YAMLContent); err != nil {
			return c.Status(400).JSON(ErrorResponse{Error: fmt.Sprintf("Invalid plugin YAML: %v", err)})
		}

		version, err := repo.CreatePluginVersion(id, req.YAMLContent)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				return c.Status(409).JSON(ErrorResponse{Error: err.Error()})
			}
			return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
		}

		return c.JSON(fiber.Map{
			"message": "New version created",
			"version": version,
		})
	}

	// Otherwise, just update metadata
	if err := repo.UpdatePlugin(id, req.Description); err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(SuccessResponse{Message: "Plugin updated successfully"})
}

// deletePlugin deletes a plugin and all its versions
func (s *Server) deletePlugin(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(ErrorResponse{Error: "Plugin ID is required"})
	}

	// TODO: Check if any active workflows are using this plugin
	// For now, we'll allow deletion

	repo := database.NewPluginRepo(s.db)
	if err := repo.DeletePlugin(id); err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(SuccessResponse{Message: "Plugin deleted successfully"})
}

// getPluginVersions returns all versions of a plugin
func (s *Server) getPluginVersions(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(ErrorResponse{Error: "Plugin ID is required"})
	}

	repo := database.NewPluginRepo(s.db)
	versions, err := repo.GetPluginVersions(id)
	if err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(versions)
}

// createPluginVersion creates a new version for a plugin
func (s *Server) createPluginVersion(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(ErrorResponse{Error: "Plugin ID is required"})
	}

	var req struct {
		YAMLContent string `json:"yaml_content"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	if req.YAMLContent == "" {
		return c.Status(400).JSON(ErrorResponse{Error: "YAML content is required"})
	}

	// Validate YAML structure
	if err := validatePluginYAML(req.YAMLContent); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: fmt.Sprintf("Invalid plugin YAML: %v", err)})
	}

	repo := database.NewPluginRepo(s.db)
	version, err := repo.CreatePluginVersion(id, req.YAMLContent)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return c.Status(409).JSON(ErrorResponse{Error: err.Error()})
		}
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.Status(201).JSON(version)
}

// activatePluginVersion sets a version as the current version
func (s *Server) activatePluginVersion(c *fiber.Ctx) error {
	pluginID := c.Params("id")
	versionID := c.Params("version_id")

	if pluginID == "" || versionID == "" {
		return c.Status(400).JSON(ErrorResponse{Error: "Plugin ID and version ID are required"})
	}

	repo := database.NewPluginRepo(s.db)
	if err := repo.SetCurrentVersion(pluginID, versionID); err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(SuccessResponse{Message: "Version activated successfully"})
}

// searchPlugins searches plugins by query, source, or tags
func (s *Server) searchPlugins(c *fiber.Ctx) error {
	query := c.Query("query", "")
	source := c.Query("source", "")
	tagsStr := c.Query("tags", "")

	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}

	repo := database.NewPluginRepo(s.db)
	plugins, err := repo.SearchPlugins(query, source, tags)
	if err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(plugins)
}

// validatePluginYAML validates the structure of a plugin YAML
func validatePluginYAML(yamlContent string) error {
	var plugin struct {
		Name         string                 `yaml:"name"`
		Description  string                 `yaml:"description"`
		Version      string                 `yaml:"version"`
		Dependencies []string               `yaml:"dependencies"`
		Inputs       map[string]interface{} `yaml:"inputs"`
		Steps        []interface{}          `yaml:"steps"`
		Tags         []string               `yaml:"tags"`
	}

	if err := yaml.Unmarshal([]byte(yamlContent), &plugin); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// Validate required fields
	if plugin.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if plugin.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	if len(plugin.Steps) == 0 {
		return fmt.Errorf("plugin must have at least one step")
	}

	// Validate version format (basic semantic versioning check)
	parts := strings.Split(plugin.Version, ".")
	if len(parts) != 3 {
		return fmt.Errorf("version must be in semantic versioning format (e.g., 1.0.0)")
	}

	return nil
}
