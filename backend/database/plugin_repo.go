package database

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// PluginRepo handles plugin database operations
type PluginRepo struct {
	db *DB
}

// NewPluginRepo creates a new plugin repository
func NewPluginRepo(db *DB) *PluginRepo {
	return &PluginRepo{db: db}
}

// CreatePlugin creates a new plugin with its first version
func (r *PluginRepo) CreatePlugin(name, description, yamlContent, createdBy string) (*Plugin, *PluginVersion, error) {
	// Parse YAML to extract version and validate structure
	var pluginDef struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal([]byte(yamlContent), &pluginDef); err != nil {
		return nil, nil, fmt.Errorf("invalid plugin YAML: %w", err)
	}

	if pluginDef.Version == "" {
		return nil, nil, fmt.Errorf("plugin YAML must include a version field")
	}

	// Create plugin
	pluginID := uuid.New().String()
	versionID := uuid.New().String()

	plugin := &PluginModel{
		ID:               pluginID,
		Name:             name,
		Description:      description,
		CurrentVersionID: versionID,
		Source:           "local",
		CreatedBy:        createdBy,
	}

	version := &PluginVersionModel{
		ID:          versionID,
		PluginID:    pluginID,
		Version:     pluginDef.Version,
		YAMLContent: yamlContent,
		Description: description,
	}

	// Create both in transaction
	err := r.db.conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(plugin).Error; err != nil {
			return err
		}
		if err := tx.Create(version).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return plugin.ToPlugin(), version.ToPluginVersion(), nil
}

// GetAllPlugins returns all plugins with their current version info
func (r *PluginRepo) GetAllPlugins() ([]*Plugin, error) {
	var plugins []PluginModel
	if err := r.db.conn.Order("name ASC").Find(&plugins).Error; err != nil {
		return nil, err
	}

	result := make([]*Plugin, len(plugins))
	for i, p := range plugins {
		plugin := p.ToPlugin()

		// Get current version info
		if p.CurrentVersionID != "" {
			var version PluginVersionModel
			if err := r.db.conn.Where("id = ?", p.CurrentVersionID).First(&version).Error; err == nil {
				plugin.CurrentVersion = version.Version
			}
		}

		result[i] = plugin
	}

	return result, nil
}

// GetPluginByID returns a plugin by ID
func (r *PluginRepo) GetPluginByID(id string) (*Plugin, error) {
	var plugin PluginModel
	if err := r.db.conn.Where("id = ?", id).First(&plugin).Error; err != nil {
		return nil, err
	}

	result := plugin.ToPlugin()

	// Get current version
	if plugin.CurrentVersionID != "" {
		var version PluginVersionModel
		if err := r.db.conn.Where("id = ?", plugin.CurrentVersionID).First(&version).Error; err == nil {
			result.CurrentVersion = version.Version
		}
	}

	return result, nil
}

// GetPluginByName returns a plugin by name
func (r *PluginRepo) GetPluginByName(name string) (*Plugin, error) {
	var plugin PluginModel
	if err := r.db.conn.Where("name = ?", name).First(&plugin).Error; err != nil {
		return nil, err
	}

	result := plugin.ToPlugin()

	// Get current version
	if plugin.CurrentVersionID != "" {
		var version PluginVersionModel
		if err := r.db.conn.Where("id = ?", plugin.CurrentVersionID).First(&version).Error; err == nil {
			result.CurrentVersion = version.Version
		}
	}

	return result, nil
}

// GetPluginWithVersions returns a plugin with all its versions
func (r *PluginRepo) GetPluginWithVersions(id string) (*PluginWithVersions, error) {
	plugin, err := r.GetPluginByID(id)
	if err != nil {
		return nil, err
	}

	versions, err := r.GetPluginVersions(id)
	if err != nil {
		return nil, err
	}

	return &PluginWithVersions{
		Plugin:   plugin,
		Versions: versions,
	}, nil
}

// GetPluginVersions returns all versions of a plugin
func (r *PluginRepo) GetPluginVersions(pluginID string) ([]*PluginVersion, error) {
	var versions []PluginVersionModel
	if err := r.db.conn.Where("plugin_id = ?", pluginID).Order("created_at DESC").Find(&versions).Error; err != nil {
		return nil, err
	}

	result := make([]*PluginVersion, len(versions))
	for i, v := range versions {
		result[i] = v.ToPluginVersion()
	}

	return result, nil
}

// GetPluginVersion returns a specific version by ID
func (r *PluginRepo) GetPluginVersionByID(versionID string) (*PluginVersion, error) {
	var version PluginVersionModel
	if err := r.db.conn.Where("id = ?", versionID).First(&version).Error; err != nil {
		return nil, err
	}
	return version.ToPluginVersion(), nil
}

// GetPluginVersionByNumber returns a specific version by plugin name and version number
func (r *PluginRepo) GetPluginVersionByNumber(pluginName, version string) (*PluginVersion, error) {
	// First get plugin
	plugin, err := r.GetPluginByName(pluginName)
	if err != nil {
		return nil, err
	}

	// Then get specific version
	var versionModel PluginVersionModel
	if err := r.db.conn.Where("plugin_id = ? AND version = ?", plugin.ID, version).First(&versionModel).Error; err != nil {
		return nil, err
	}

	return versionModel.ToPluginVersion(), nil
}

// GetPluginCurrentVersion returns the current/latest version of a plugin
func (r *PluginRepo) GetPluginCurrentVersion(pluginID string) (*PluginVersion, error) {
	var plugin PluginModel
	if err := r.db.conn.Where("id = ?", pluginID).First(&plugin).Error; err != nil {
		return nil, err
	}

	if plugin.CurrentVersionID == "" {
		return nil, fmt.Errorf("plugin has no current version")
	}

	return r.GetPluginVersionByID(plugin.CurrentVersionID)
}

// CreatePluginVersion creates a new version for an existing plugin
func (r *PluginRepo) CreatePluginVersion(pluginID, yamlContent string) (*PluginVersion, error) {
	// Parse YAML to extract version
	var pluginDef struct {
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(yamlContent), &pluginDef); err != nil {
		return nil, fmt.Errorf("invalid plugin YAML: %w", err)
	}

	if pluginDef.Version == "" {
		return nil, fmt.Errorf("plugin YAML must include a version field")
	}

	// Check if version already exists
	var existingCount int64
	if err := r.db.conn.Model(&PluginVersionModel{}).
		Where("plugin_id = ? AND version = ?", pluginID, pluginDef.Version).
		Count(&existingCount).Error; err != nil {
		return nil, err
	}

	if existingCount > 0 {
		return nil, fmt.Errorf("version %s already exists for this plugin", pluginDef.Version)
	}

	versionID := uuid.New().String()
	version := &PluginVersionModel{
		ID:          versionID,
		PluginID:    pluginID,
		Version:     pluginDef.Version,
		YAMLContent: yamlContent,
		Description: pluginDef.Description,
	}

	// Create version and update plugin's current version in transaction
	err := r.db.conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(version).Error; err != nil {
			return err
		}
		// Update plugin's current version to the new version
		if err := tx.Model(&PluginModel{}).Where("id = ?", pluginID).
			Update("current_version_id", versionID).Error; err != nil {
			return err
		}
		if err := tx.Model(&PluginModel{}).Where("id = ?", pluginID).
			Update("updated_at", time.Now()).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return version.ToPluginVersion(), nil
}

// SetCurrentVersion sets a specific version as the current version for a plugin
func (r *PluginRepo) SetCurrentVersion(pluginID, versionID string) error {
	// Verify version belongs to plugin
	var version PluginVersionModel
	if err := r.db.conn.Where("id = ? AND plugin_id = ?", versionID, pluginID).First(&version).Error; err != nil {
		return fmt.Errorf("version not found or does not belong to plugin: %w", err)
	}

	return r.db.conn.Model(&PluginModel{}).Where("id = ?", pluginID).
		Updates(map[string]interface{}{
			"current_version_id": versionID,
			"updated_at":         time.Now(),
		}).Error
}

// UpdatePlugin updates plugin metadata (not version content)
func (r *PluginRepo) UpdatePlugin(id, description string) error {
	return r.db.conn.Model(&PluginModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"description": description,
			"updated_at":  time.Now(),
		}).Error
}

// DeletePlugin deletes a plugin and all its versions
func (r *PluginRepo) DeletePlugin(id string) error {
	return r.db.conn.Transaction(func(tx *gorm.DB) error {
		// Delete all versions
		if err := tx.Where("plugin_id = ?", id).Delete(&PluginVersionModel{}).Error; err != nil {
			return err
		}
		// Delete plugin
		if err := tx.Where("id = ?", id).Delete(&PluginModel{}).Error; err != nil {
			return err
		}
		return nil
	})
}

// SearchPlugins searches plugins by name, source, or tags
func (r *PluginRepo) SearchPlugins(query, source string, tags []string) ([]*Plugin, error) {
	var plugins []PluginModel

	db := r.db.conn

	// Filter by source if provided
	if source != "" {
		db = db.Where("source = ?", source)
	}

	// Filter by name if query provided
	if query != "" {
		db = db.Where("name LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%")
	}

	if err := db.Order("name ASC").Find(&plugins).Error; err != nil {
		return nil, err
	}

	result := make([]*Plugin, 0)
	for _, p := range plugins {
		plugin := p.ToPlugin()

		// Get current version info
		if p.CurrentVersionID != "" {
			var version PluginVersionModel
			if err := r.db.conn.Where("id = ?", p.CurrentVersionID).First(&version).Error; err == nil {
				plugin.CurrentVersion = version.Version

				// Parse tags from YAML if tag filter is provided
				if len(tags) > 0 {
					var pluginDef struct {
						Tags []string `yaml:"tags"`
					}
					if err := yaml.Unmarshal([]byte(version.YAMLContent), &pluginDef); err == nil {
						plugin.Tags = pluginDef.Tags

						// Check if plugin has any of the requested tags
						hasTag := false
						for _, requestedTag := range tags {
							for _, pluginTag := range pluginDef.Tags {
								if pluginTag == requestedTag {
									hasTag = true
									break
								}
							}
							if hasTag {
								break
							}
						}

						if !hasTag {
							continue // Skip this plugin
						}
					}
				}
			}
		}

		result = append(result, plugin)
	}

	return result, nil
}
