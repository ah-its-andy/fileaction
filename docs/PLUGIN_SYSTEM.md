# Plugin System Documentation

## Overview

The FileAction plugin system allows you to create reusable, versioned workflow components that can be shared and integrated into workflows. Plugins encapsulate complex operations into simple, configurable steps.

## Plugin Structure

A plugin is defined using YAML format with the following structure:

```yaml
name: plugin-name
description: Brief description of what the plugin does
version: 1.0.0
dependencies:
  - command1
  - command2>=version
inputs:
  input_name:
    type: string|number|boolean
    default: default_value
    required: true|false
    description: Input description
steps:
  - name: Step name
    run: shell command
    condition: optional condition
    timeout: timeout in seconds
    env:
      VAR_NAME: value
tags:
  - tag1
  - tag2
```

### Required Fields

- **name**: Unique identifier for the plugin (lowercase, alphanumeric, hyphens, underscores)
- **version**: Semantic version number (e.g., "1.0.0")
- **steps**: Array of at least one step to execute

### Optional Fields

- **description**: Human-readable description
- **dependencies**: List of required system commands/tools
- **inputs**: Configuration parameters for the plugin
- **tags**: Categories for organizing plugins
- **env**: Global environment variables for all steps

## Using Plugins in Workflows

### Basic Usage

Include a plugin in your workflow using the `uses` directive:

```yaml
name: My Workflow
on:
  paths:
    - /path/to/watch
steps:
  - name: Optimize Images
    uses: image-optimizer@v1.0.0
    with:
      quality: 85
      strip_metadata: true
```

### Version Specification

- **Specific version**: `plugin-name@v1.0.0`
- **Latest version**: `plugin-name` (omit version)

### Providing Inputs

Use the `with` directive to pass input values:

```yaml
steps:
  - name: Transcode Video
    uses: video-transcoder@v1.0.0
    with:
      output_format: mp4
      resolution: 1920x1080
      bitrate: 2M
```

## Input Parameters

### Input Types

- **string**: Text value
- **number**: Numeric value
- **boolean**: true/false value

### Input Properties

- **type**: Data type of the input
- **default**: Default value if not provided
- **required**: Whether input must be provided
- **description**: Help text for the input

## Variable Substitution

### Workflow Variables

Available in plugin steps:

- `${{ input_path }}`: Full path to input file
- `${{ output_path }}`: Full path to output file
- `${{ file_name }}`: Name of the file
- `${{ file_dir }}`: Directory containing the file
- `${{ file_base }}`: Filename without extension
- `${{ file_ext }}`: File extension

### Plugin Inputs

Access plugin input values:

- `${{ inputs.input_name }}`: Value of the input parameter

### Example

```yaml
steps:
  - name: Process File
    run: |
      echo "Processing ${{ file_name }}"
      convert "${{ input_path }}" \
        -quality ${{ inputs.quality }} \
        "${{ output_path }}"
```

## Conditional Execution

Steps can include conditions to control when they execute:

```yaml
steps:
  - name: Optimize JPEG
    condition: ${{ file_ext == '.jpg' || file_ext == '.jpeg' }}
    run: jpegoptim "${{ input_path }}"
  
  - name: Optimize PNG
    condition: ${{ file_ext == '.png' }}
    run: optipng "${{ input_path }}"
```

### Condition Syntax

- Equality: `${{ var == 'value' }}`
- Inequality: `${{ var != 'value' }}`
- Boolean: `${{ inputs.enabled == 'true' }}`

## Dependencies

Specify required system commands:

```yaml
dependencies:
  - ffmpeg        # Just check if command exists
  - node>=14.0.0  # Check for minimum version (planned)
  - python>=3.8   # Check for minimum version (planned)
```

The runtime will validate dependencies before executing plugin steps.

## Environment Variables

### Global Environment

Set environment variables for all steps:

```yaml
env:
  TMPDIR: /tmp/processing
  LOG_LEVEL: debug
```

### Step-Specific Environment

Override or add environment variables per step:

```yaml
steps:
  - name: Process with custom settings
    run: process_command
    env:
      QUALITY: high
      THREADS: 4
```

### Environment Priority

From lowest to highest priority:
1. System environment
2. Workflow environment
3. Plugin global environment
4. Plugin step environment

## Step Configuration

### Timeout

Specify maximum execution time in seconds:

```yaml
steps:
  - name: Long Running Task
    run: process_large_file
    timeout: 3600  # 1 hour
```

### Exit Codes

- **0**: Success, continue to next step
- **100**: Success, stop workflow (workflow succeeds)
- **101**: Failure, stop workflow (workflow fails)
- **Other non-zero**: Step failure (workflow fails)

## Version Management

### Creating New Versions

When you edit a plugin through the UI or API, a new version is automatically created. The version number should follow semantic versioning:

- **Major** (1.0.0 → 2.0.0): Breaking changes
- **Minor** (1.0.0 → 1.1.0): New features, backward compatible
- **Patch** (1.0.0 → 1.0.1): Bug fixes

### Version Rollback

You can activate any previous version through the UI or API. This makes that version the "current" version for new workflow executions.

### Version Pinning

Workflows can pin to specific versions:

```yaml
steps:
  - name: Use specific version
    uses: my-plugin@v1.2.3
    
  - name: Always use latest
    uses: my-plugin
```

## Example Plugins

### 1. Image Optimizer

```yaml
name: image-optimizer
description: Optimize images using various compression tools
version: 1.0.0
dependencies:
  - jpegoptim
  - optipng
inputs:
  quality:
    type: number
    default: 85
    required: false
    description: "JPEG quality (1-100)"
steps:
  - name: Optimize JPEG
    condition: ${{ file_ext == '.jpg' }}
    run: jpegoptim --max=${{ inputs.quality }} "${{ input_path }}"
  
  - name: Optimize PNG
    condition: ${{ file_ext == '.png' }}
    run: optipng -o5 "${{ input_path }}"
tags:
  - image
  - optimization
```

### 2. Notification Sender

```yaml
name: notification-sender
description: Send notifications via webhook
version: 1.0.0
dependencies:
  - curl
inputs:
  webhook_url:
    type: string
    default: ""
    required: true
    description: "Webhook URL"
  message:
    type: string
    default: "Processing completed"
    required: false
    description: "Notification message"
steps:
  - name: Send notification
    run: |
      curl -X POST "${{ inputs.webhook_url }}" \
        -H "Content-Type: application/json" \
        -d '{"message":"${{ inputs.message }}","file":"${{ file_name }}"}'
tags:
  - notification
  - webhook
```

## Best Practices

### 1. Clear Naming

Use descriptive, lowercase names with hyphens:
- ✅ `image-optimizer`
- ✅ `pdf-generator`
- ❌ `ImageOptimizer`
- ❌ `img_opt`

### 2. Semantic Versioning

Follow semantic versioning strictly:
- Breaking changes → major version bump
- New features → minor version bump
- Bug fixes → patch version bump

### 3. Input Validation

Always validate required inputs:

```yaml
steps:
  - name: Validate inputs
    run: |
      if [ -z "${{ inputs.required_param }}" ]; then
        echo "Error: required_param is missing"
        exit 1
      fi
```

### 4. Error Handling

Provide clear error messages:

```yaml
steps:
  - name: Process file
    run: |
      if [ ! -f "${{ input_path }}" ]; then
        echo "Error: Input file not found: ${{ input_path }}"
        exit 1
      fi
      # Process file...
```

### 5. Documentation

Include comprehensive descriptions for inputs and the plugin itself.

### 6. Idempotency

Design plugins to be idempotent when possible - running them multiple times should produce the same result.

### 7. Resource Cleanup

Clean up temporary files and resources:

```yaml
steps:
  - name: Cleanup
    run: |
      rm -rf /tmp/processing/*
      echo "Cleanup complete"
```

## API Integration

### Create Plugin

```bash
curl -X POST http://localhost:3000/api/plugins \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-plugin",
    "description": "My custom plugin",
    "yaml_content": "..."
  }'
```

### Update Plugin (Creates New Version)

```bash
curl -X PUT http://localhost:3000/api/plugins/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "yaml_content": "..."
  }'
```

### Activate Version

```bash
curl -X PUT http://localhost:3000/api/plugins/{id}/versions/{version_id}/activate
```

### Search Plugins

```bash
curl "http://localhost:3000/api/plugins/search?query=image&source=local&tags=optimization"
```

## Troubleshooting

### Plugin Not Found

- Verify plugin name matches exactly
- Check if plugin exists in the Plugins tab
- Ensure version exists if specified

### Dependency Not Found

- Install required dependencies on the server
- Verify command is in PATH
- Check dependency name spelling

### Input Validation Errors

- Ensure all required inputs are provided
- Check input types match specification
- Verify default values are appropriate

### Step Execution Failures

- Check step logs for error messages
- Verify variable substitution is correct
- Test commands manually on the server
- Check file permissions and paths

## Future Enhancements

- Plugin marketplace for sharing
- Git repository integration for plugin sync
- Visual plugin editor
- Plugin testing framework
- Dependency version checking
- Plugin composition (plugins using other plugins)
