package workflow

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// PluginDef represents a parsed plugin definition
type PluginDef struct {
	Name         string                 `yaml:"name"`
	Description  string                 `yaml:"description"`
	Version      string                 `yaml:"version"`
	Dependencies []string               `yaml:"dependencies"`
	Inputs       map[string]PluginInput `yaml:"inputs"`
	Steps        []PluginStep           `yaml:"steps"`
	Tags         []string               `yaml:"tags"`
	Env          map[string]string      `yaml:"env"`
}

// PluginInput represents an input parameter for a plugin
type PluginInput struct {
	Type        string      `yaml:"type"`
	Default     interface{} `yaml:"default"`
	Required    bool        `yaml:"required"`
	Description string      `yaml:"description"`
}

// PluginStep represents a step within a plugin
type PluginStep struct {
	Name      string            `yaml:"name"`
	Run       string            `yaml:"run"`
	Condition string            `yaml:"condition"`
	Timeout   int               `yaml:"timeout"` // In seconds
	Env       map[string]string `yaml:"env"`
}

// ParsePlugin parses a plugin YAML definition
func ParsePlugin(yamlContent string) (*PluginDef, error) {
	var plugin PluginDef
	if err := yaml.Unmarshal([]byte(yamlContent), &plugin); err != nil {
		return nil, fmt.Errorf("failed to parse plugin YAML: %w", err)
	}

	// Validate required fields
	if plugin.Name == "" {
		return nil, fmt.Errorf("plugin name is required")
	}
	if plugin.Version == "" {
		return nil, fmt.Errorf("plugin version is required")
	}
	if len(plugin.Steps) == 0 {
		return nil, fmt.Errorf("plugin must have at least one step")
	}

	return &plugin, nil
}

// ParsePluginReference parses a plugin reference string (e.g., "plugin_name@v1.0.0")
// Returns plugin name and version (empty string means use latest)
func ParsePluginReference(uses string) (string, string, error) {
	if uses == "" {
		return "", "", fmt.Errorf("empty plugin reference")
	}

	parts := strings.Split(uses, "@")
	if len(parts) == 1 {
		// No version specified, use latest
		return parts[0], "", nil
	}
	if len(parts) == 2 {
		pluginName := parts[0]
		version := strings.TrimPrefix(parts[1], "v") // Remove 'v' prefix if present
		return pluginName, version, nil
	}

	return "", "", fmt.Errorf("invalid plugin reference format: %s", uses)
}

// ValidatePluginDependencies checks if all required dependencies are available
func ValidatePluginDependencies(dependencies []string) error {
	for _, dep := range dependencies {
		// Parse dependency (format: "command" or "command>=version")
		parts := strings.FieldsFunc(dep, func(r rune) bool {
			return r == '>' || r == '<' || r == '='
		})

		if len(parts) == 0 {
			continue
		}

		command := strings.TrimSpace(parts[0])

		// Check if command exists
		_, err := exec.LookPath(command)
		if err != nil {
			return fmt.Errorf("required dependency '%s' not found", command)
		}

		// TODO: Implement version checking if version constraint is specified
		// For now, we just check if the command exists
	}

	return nil
}

// SubstitutePluginInputs replaces input placeholders in a command string
// Supports formats: ${{ inputs.param_name }} or ${{ input.param_name }}
func SubstitutePluginInputs(command string, inputs map[string]string) string {
	result := command

	// Pattern to match ${{ inputs.param_name }} or ${{ input.param_name }}
	re := regexp.MustCompile(`\$\{\{\s*inputs?\.(\w+)\s*\}\}`)

	result = re.ReplaceAllStringFunc(result, func(match string) string {
		// Extract parameter name
		matches := re.FindStringSubmatch(match)
		if len(matches) > 1 {
			paramName := matches[1]
			if value, ok := inputs[paramName]; ok {
				return value
			}
		}
		return match // Return original if not found
	})

	return result
}

// PreparePluginInputs merges default values with provided values
func PreparePluginInputs(pluginDef *PluginDef, providedInputs map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	// First, set all defaults
	for name, input := range pluginDef.Inputs {
		if input.Default != nil {
			result[name] = fmt.Sprintf("%v", input.Default)
		}
	}

	// Then override with provided values
	for name, value := range providedInputs {
		result[name] = value
	}

	// Validate required inputs
	for name, input := range pluginDef.Inputs {
		if input.Required {
			if _, ok := result[name]; !ok {
				return nil, fmt.Errorf("required input '%s' is missing", name)
			}
		}
	}

	return result, nil
}

// EvaluateCondition evaluates a simple condition expression
// Supports basic comparisons like: "${{ inputs.enabled == 'true' }}"
func EvaluateCondition(condition string, inputs map[string]string, vars Variables) bool {
	if condition == "" {
		return true // No condition means always execute
	}

	// Substitute inputs first
	condition = SubstitutePluginInputs(condition, inputs)

	// Substitute workflow variables
	condition = SubstituteVariables(condition, vars)

	// Remove ${{ }} wrapper if present
	condition = strings.TrimSpace(condition)
	condition = strings.TrimPrefix(condition, "${{")
	condition = strings.TrimSuffix(condition, "}}")
	condition = strings.TrimSpace(condition)

	// Simple equality check
	if strings.Contains(condition, "==") {
		parts := strings.Split(condition, "==")
		if len(parts) == 2 {
			left := strings.TrimSpace(strings.Trim(parts[0], "'\""))
			right := strings.TrimSpace(strings.Trim(parts[1], "'\""))
			return left == right
		}
	}

	// Simple inequality check
	if strings.Contains(condition, "!=") {
		parts := strings.Split(condition, "!=")
		if len(parts) == 2 {
			left := strings.TrimSpace(strings.Trim(parts[0], "'\""))
			right := strings.TrimSpace(strings.Trim(parts[1], "'\""))
			return left != right
		}
	}

	// Boolean check (treat non-empty, non-false as true)
	condition = strings.ToLower(condition)
	return condition != "" && condition != "false" && condition != "0"
}

// MergeEnvironment merges multiple environment variable maps
// Priority: stepEnv > pluginEnv > workflowEnv > baseEnv
func MergeEnvironment(baseEnv, workflowEnv, pluginEnv, stepEnv map[string]string) map[string]string {
	result := make(map[string]string)

	// Copy in order of priority (lowest to highest)
	for k, v := range baseEnv {
		result[k] = v
	}
	for k, v := range workflowEnv {
		result[k] = v
	}
	for k, v := range pluginEnv {
		result[k] = v
	}
	for k, v := range stepEnv {
		result[k] = v
	}

	return result
}
