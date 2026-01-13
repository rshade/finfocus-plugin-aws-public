# Data Model - Rename Plugin to FinFocus

**Feature Branch:** `001-rename-plugin-finfocus`

## Overview

The underlying data model for cost estimation remains unchanged. This feature purely renames the identity of the plugin and its configuration surface.

## Configuration Entities

| Entity | Old Name (Deprecated) | New Name (Standard) | Notes |
|--------|-----------------------|---------------------|-------|
| **Module Name** | `finfocus-plugin-aws-public` | `finfocus-plugin-aws-public` | Go module identity |
| **Binary Name** | `finfocus-plugin-aws-public-<region>` | `finfocus-plugin-aws-public-<region>` | Executable name |
| **Log Component** | `[finfocus-plugin-aws-public]` | `[finfocus-plugin-aws-public]` | Logging prefix |

## Environment Variables

| Variable | Legacy Key | New Key | Behavior |
|----------|------------|---------|----------|
| **Test Mode** | `FINFOCUS_TEST_MODE` | `FINFOCUS_TEST_MODE` | New key takes precedence. Legacy key logs warning. |
| **Batch Size** | `MAX_BATCH_SIZE` | `FINFOCUS_MAX_BATCH_SIZE` | New key takes precedence. |
| **Strict Validation** | `STRICT_VALIDATION` | `FINFOCUS_STRICT_VALIDATION` | New key takes precedence. |

*Note: The system will also support `FINFOCUS_` prefixes for batch size and validation if implemented.*
