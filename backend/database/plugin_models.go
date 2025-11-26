package database

import (
	"time"
)

// PluginModel represents a plugin in the database
type PluginModel struct {
	ID               string    `gorm:"primaryKey;type:varchar(36)"`
	Name             string    `gorm:"uniqueIndex;type:varchar(255);not null"`
	Description      string    `gorm:"type:text"`
	CurrentVersionID string    `gorm:"type:varchar(36);index"`                    // Points to the current active version
	Source           string    `gorm:"type:varchar(50);not null;default:'local'"` // 'local' for now, future: 'git', 'marketplace'
	CreatedBy        string    `gorm:"type:varchar(255)"`
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
}

func (PluginModel) TableName() string {
	return "plugins"
}

// PluginVersionModel represents a specific version of a plugin
type PluginVersionModel struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)"`
	PluginID    string    `gorm:"type:varchar(36);not null;index"`
	Version     string    `gorm:"type:varchar(50);not null"` // Semantic version (e.g., "1.0.0")
	YAMLContent string    `gorm:"type:text;not null"`        // Full YAML definition
	Description string    `gorm:"type:text"`                 // Version-specific description
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (PluginVersionModel) TableName() string {
	return "plugin_versions"
}

// ToPlugin converts PluginModel to models.Plugin
func (m *PluginModel) ToPlugin() *Plugin {
	return &Plugin{
		ID:               m.ID,
		Name:             m.Name,
		Description:      m.Description,
		CurrentVersionID: m.CurrentVersionID,
		Source:           m.Source,
		CreatedBy:        m.CreatedBy,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

// FromPlugin converts models.Plugin to PluginModel
func FromPlugin(p *Plugin) *PluginModel {
	return &PluginModel{
		ID:               p.ID,
		Name:             p.Name,
		Description:      p.Description,
		CurrentVersionID: p.CurrentVersionID,
		Source:           p.Source,
		CreatedBy:        p.CreatedBy,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

// ToPluginVersion converts PluginVersionModel to models.PluginVersion
func (m *PluginVersionModel) ToPluginVersion() *PluginVersion {
	return &PluginVersion{
		ID:          m.ID,
		PluginID:    m.PluginID,
		Version:     m.Version,
		YAMLContent: m.YAMLContent,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
	}
}

// FromPluginVersion converts models.PluginVersion to PluginVersionModel
func FromPluginVersion(pv *PluginVersion) *PluginVersionModel {
	return &PluginVersionModel{
		ID:          pv.ID,
		PluginID:    pv.PluginID,
		Version:     pv.Version,
		YAMLContent: pv.YAMLContent,
		Description: pv.Description,
		CreatedAt:   pv.CreatedAt,
	}
}

// Plugin represents a plugin (business logic model)
type Plugin struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	CurrentVersionID string    `json:"current_version_id"`
	CurrentVersion   string    `json:"current_version,omitempty"` // Populated from version lookup
	Source           string    `json:"source"`
	Tags             []string  `json:"tags,omitempty"` // Parsed from YAML
	CreatedBy        string    `json:"created_by,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// PluginVersion represents a specific version of a plugin
type PluginVersion struct {
	ID          string    `json:"id"`
	PluginID    string    `json:"plugin_id"`
	Version     string    `json:"version"`
	YAMLContent string    `json:"yaml_content"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// PluginWithVersions combines plugin with all its versions
type PluginWithVersions struct {
	Plugin   *Plugin          `json:"plugin"`
	Versions []*PluginVersion `json:"versions"`
}
