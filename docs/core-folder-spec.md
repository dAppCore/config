# .core/ Folder Specification

This document defines the `.core/` folder structure used across Host UK packages for configuration, tooling integration, and development environment setup.

## Overview

The `.core/` folder provides a standardised location for:
- Build and development configuration
- Claude Code plugin integration
- VM/container definitions
- Development environment settings

## Directory Structure

```
package/.core/
├── config.yaml           # Build targets, test commands, deploy config
├── workspace.yaml        # Workspace-level config (devops repo only)
├── plugin/               # Claude Code integration
│   ├── plugin.json       # Plugin manifest
│   ├── skills/           # Context-aware skills
│   └── hooks/            # Pre/post command hooks
├── linuxkit/             # VM/container definitions (if applicable)
│   ├── kernel.yaml
│   └── image.yaml
└── run.yaml              # Development environment config
```

## Configuration Files

### config.yaml

Package-level build and runtime configuration.

```yaml
version: 1

# Build configuration
build:
  targets:
    - name: default
      command: composer build
    - name: production
      command: composer build:prod
      env:
        APP_ENV: production

# Test configuration
test:
  command: composer test
  coverage: true
  parallel: true

# Lint configuration
lint:
  command: ./vendor/bin/pint
  fix_command: ./vendor/bin/pint --dirty

# Deploy configuration (if applicable)
deploy:
  staging:
    command: ./deploy.sh staging
  production:
    command: ./deploy.sh production
    requires_approval: true
```

### workspace.yaml

Workspace-level configuration (only in `core-devops`).

```yaml
version: 1

# Active package for unified commands
active: core-php

# Default package types for setup
default_only:
  - foundation
  - module

# Paths
packages_dir: ./packages

# Workspace settings
settings:
  suggest_core_commands: true
  show_active_in_prompt: true
```

### run.yaml

Development environment configuration.

```yaml
version: 1

# Services required for development
services:
  - name: database
    image: postgres:16
    port: 5432
    env:
      POSTGRES_DB: core_dev
      POSTGRES_USER: core
      POSTGRES_PASSWORD: secret

  - name: redis
    image: redis:7
    port: 6379

  - name: mailpit
    image: axllent/mailpit
    port: 8025

# Development server
dev:
  command: php artisan serve
  port: 8000
  watch:
    - app/
    - resources/

# Environment variables
env:
  APP_ENV: local
  APP_DEBUG: true
  DB_CONNECTION: pgsql
```

## Claude Code Plugin

### plugin.json

The plugin manifest defines skills, hooks, and commands for Claude Code integration.

```json
{
  "$schema": "https://claude.ai/code/plugin-schema.json",
  "name": "package-name",
  "version": "1.0.0",
  "description": "Claude Code integration for this package",

  "skills": [
    {
      "name": "skill-name",
      "file": "skills/skill-name.md",
      "description": "What this skill provides"
    }
  ],

  "hooks": {
    "pre_command": [
      {
        "pattern": "^command-pattern$",
        "script": "hooks/script.sh",
        "description": "What this hook does"
      }
    ]
  },

  "commands": {
    "command-name": {
      "description": "What this command does",
      "run": "actual-command"
    }
  }
}
```

### Skills (skills/*.md)

Markdown files providing context-aware guidance for Claude Code. Skills are loaded when relevant to the user's query.

```markdown
# Skill Name

Describe what this skill provides.

## Context

When to use this skill.

## Commands

Relevant commands and examples.

## Tips

Best practices and gotchas.
```

### Hooks (hooks/*.sh)

Shell scripts executed before or after commands. Hooks should:
- Be executable (`chmod +x`)
- Exit 0 for informational hooks (don't block)
- Exit non-zero to block the command (with reason)

```bash
#!/bin/bash
set -euo pipefail

# Hook logic here

exit 0  # Don't block
```

## LinuxKit (linuxkit/)

For packages that deploy as VMs or containers.

### kernel.yaml

```yaml
kernel:
  image: linuxkit/kernel:6.6
  cmdline: "console=tty0"
```

### image.yaml

```yaml
image:
  - linuxkit/init:v1.0.1
  - linuxkit/runc:v1.0.0
  - linuxkit/containerd:v1.0.0
```

## Package-Type Specific Patterns

### Foundation (core-php)

```
core-php/.core/
├── config.yaml           # Build targets for framework
├── plugin/
│   └── skills/
│       ├── events.md     # Event system guidance
│       ├── modules.md    # Module loading patterns
│       └── lifecycle.md  # Lifecycle events
└── run.yaml              # Test environment setup
```

### Module (core-tenant, core-admin, etc.)

```
core-tenant/.core/
├── config.yaml           # Module-specific build
├── plugin/
│   └── skills/
│       └── tenancy.md    # Multi-tenancy patterns
└── run.yaml              # Required services (database)
```

### Product (core-bio, core-social, etc.)

```
core-bio/.core/
├── config.yaml           # Build and deploy targets
├── plugin/
│   └── skills/
│       └── bio.md        # Product-specific guidance
├── linuxkit/             # VM definitions for deployment
│   ├── kernel.yaml
│   └── image.yaml
└── run.yaml              # Full dev environment
```

### Workspace (core-devops)

```
core-devops/.core/
├── workspace.yaml        # Active package, paths
├── plugin/
│   ├── plugin.json
│   └── skills/
│       ├── workspace.md      # Multi-repo navigation
│       ├── switch-package.md # Package switching
│       └── package-status.md # Status checking
└── docs/
    └── core-folder-spec.md   # This file
```

## Core CLI Integration

The `core` CLI reads configuration from `.core/`:

| File | CLI Command | Purpose |
|------|-------------|---------|
| `workspace.yaml` | `core workspace` | Active package, paths |
| `config.yaml` | `core build`, `core test` | Build/test commands |
| `run.yaml` | `core run` | Dev environment |

## Best Practices

1. **Always include `version: 1`** in YAML files for future compatibility
2. **Keep skills focused** - one concept per skill file
3. **Hooks should be fast** - don't slow down commands
4. **Use relative paths** - avoid hardcoded absolute paths
5. **Document non-obvious settings** with inline comments

## Migration Guide

To add `.core/` to an existing package:

1. Create the directory structure:
   ```bash
   mkdir -p .core/plugin/skills .core/plugin/hooks
   ```

2. Add `config.yaml` with build/test commands

3. Add `plugin.json` with package-specific skills

4. Add relevant skills in `skills/`

5. Update `.gitignore` if needed (don't ignore `.core/`)
