# Data Model: Multi-Region Router Plugin

**Branch**: `036-multi-region-router` | **Date**: 2026-02-12

## Entities

### RouterPlugin

The top-level struct implementing `pluginsdk.Plugin`. Contains no pricing data.

| Field | Type | Description |
|-------|------|-------------|
| version | string | Router binary version (from ldflags) |
| logger | zerolog.Logger | Structured logger for router events |
| registry | *ChildRegistry | Thread-safe child process registry |
| downloader | *Downloader | Binary download + verification component |
| offline | bool | True if `FINFOCUS_PLUGIN_OFFLINE=true` |
| binaryDir | string | Directory containing router and region binaries |

**Relationships**: Owns one `ChildRegistry` and one `Downloader`.

### ChildRegistry

Thread-safe map of region to child process state.

| Field | Type | Description |
|-------|------|-------------|
| mu | sync.RWMutex | Protects children map |
| children | map[string]*ChildProcess | Region → child mapping |

**Operations**:

- `GetOrLaunch(region) (*ChildProcess, error)` — Returns existing child or
  launches a new one (concurrent-safe).
- `Get(region) *ChildProcess` — Returns child if exists, nil otherwise.
- `ShutdownAll(ctx)` — Sends SIGTERM to all children, waits for exit.

### ChildProcess

Represents a running region-specific plugin child.

| Field | Type | Description |
|-------|------|-------------|
| region | string | AWS region (e.g., "us-east-1") |
| binaryPath | string | Absolute path to region binary |
| cmd | *exec.Cmd | Running process handle |
| port | int | Child's announced PORT |
| client | *pluginsdk.Client | Connect client for RPC delegation |
| state | ChildState | Current lifecycle state |
| mu | sync.Mutex | Protects state transitions |
| version | string | Child binary version (if detectable) |

**State Transitions**:

```text
         ┌──────────┐
         │  idle     │ (binary found, not started)
         └────┬─────┘
              │ first request
              ▼
         ┌──────────┐
         │ launching │ (exec.Command started, waiting for PORT)
         └────┬─────┘
              │ PORT captured
              ▼
         ┌──────────┐
         │  ready    │ (accepting RPCs via Connect client)
         └────┬─────┘
              │ RPC failure / process exit
              ▼
         ┌──────────┐
         │ unhealthy │ (needs restart on next request)
         └────┬─────┘
              │ next request (up to 3 retries)
              ▼
         ┌──────────┐
         │  ready    │ (restarted successfully)
         └──────────┘
              │ 3 retries exhausted
              ▼
         ┌──────────┐
         │  failed   │ (returns error until manually resolved)
         └──────────┘
```

### ChildState (enum)

| Value | Description |
|-------|-------------|
| ChildStateIdle | Binary discovered, not yet started |
| ChildStateLaunching | Process starting, waiting for PORT |
| ChildStateReady | Healthy, accepting RPCs |
| ChildStateUnhealthy | Process crashed or RPC failed |
| ChildStateFailed | Exceeded retry limit |

### Downloader

Handles binary download and SHA256 verification.

| Field | Type | Description |
|-------|------|-------------|
| version | string | Router version (determines release tag) |
| baseURL | string | GitHub Releases base URL (hardcoded) |
| targetDir | string | Directory to save downloaded binaries |
| checksums | map[string]string | Cached checksums from `checksums.txt` |
| httpClient | *http.Client | HTTP client for downloads |
| mu | sync.Mutex | Protects concurrent downloads |

**Operations**:

- `Download(region, os, arch) (string, error)` — Downloads, verifies, extracts
  region binary. Returns path to extracted binary.
- `fetchChecksums() error` — Downloads and caches `checksums.txt`.
- `verify(filepath, expectedHash) error` — SHA256 verification.

### Discovery

Scans the binary directory for pre-installed region binaries.

**Operations**:

- `Discover(dir string) map[string]string` — Returns map of region → binary
  path by scanning for files matching `finfocus-plugin-aws-public-{region}`.

## Validation Rules

| Rule | Enforcement |
|------|-------------|
| Region must be non-empty for cost RPCs | Router returns `InvalidArgument` |
| Region must be a valid AWS region | Delegated to child (child validates) |
| Binary must pass SHA256 verification | Downloader rejects on mismatch |
| Max 12 concurrent children | Registry enforces (one per supported region) |
| Child PORT within 30 seconds | Launch timeout returns error |
| Max 3 startup retries per request | Registry tracks retry count |
