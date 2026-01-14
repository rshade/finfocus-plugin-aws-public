# Feature Specification: Expose CORS configuration via environment variables

**Feature Branch**: `031-expose-cors-config`
**Created**: 2026-01-13
**Status**: Draft

## Clarifications

### Session 2026-01-13
- Q: Conflict Behavior (Wildcard + Credentials) → A: Log a fatal error and exit (Fail Fast).
- Q: Health Endpoint Method → A: GET only.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Configure Cross-Origin Access for Web Frontend (Priority: P1)

A developer or administrator wants to allow their web-based frontend application (hosted on a different domain) to communicate directly with the FinFocus AWS Public plugin.

**Why this priority**: Without this, browser-based integrations are impossible due to browser security policies (CORS).

**Independent Test**: Can be tested by starting the plugin with `FINFOCUS_CORS_ALLOWED_ORIGINS` set and making a cross-origin request using `curl` or a browser console.

**Acceptance Scenarios**:

1. **Given** the plugin is configured with `FINFOCUS_CORS_ALLOWED_ORIGINS=http://localhost:3000`, **When** a request arrives with `Origin: http://localhost:3000`, **Then** the response includes `Access-Control-Allow-Origin: http://localhost:3000`.
2. **Given** the plugin is configured with `FINFOCUS_CORS_ALLOWED_ORIGINS=http://localhost:3000`, **When** a request arrives with `Origin: http://evil.com`, **Then** the response does NOT include CORS headers for that origin.
3. **Given** the plugin is configured with `FINFOCUS_CORS_ALLOWED_ORIGINS=*`, **When** the plugin starts, **Then** a warning log is emitted about insecure wildcard configuration.

---

### User Story 2 - Enable Credentials for Authenticated Requests (Priority: P2)

An administrator needs to support requests that include credentials (cookies, authorization headers) from the frontend.

**Why this priority**: Required for authenticated sessions or specific security contexts, though the public plugin may be less sensitive, integration with broader systems often requires this.

**Independent Test**: Start plugin with credentials enabled and verify `Access-Control-Allow-Credentials` header.

**Acceptance Scenarios**:

1. **Given** `FINFOCUS_CORS_ALLOW_CREDENTIALS=true` and a specific allowed origin, **When** a valid request arrives, **Then** the response includes `Access-Control-Allow-Credentials: true`.
2. **Given** `FINFOCUS_CORS_ALLOW_CREDENTIALS=true` and `FINFOCUS_CORS_ALLOWED_ORIGINS=*`, **When** the plugin starts or receives a request, **Then** the configuration is treated as invalid or secure default is enforced (credentials cannot be true with wildcard origin).

---

### User Story 3 - Enable Health Check Endpoint (Priority: P2)

An infrastructure engineer wants to configure a readiness/liveness probe for the plugin in a containerized environment (e.g., Kubernetes).

**Why this priority**: Essential for reliable deployment and orchestration.

**Independent Test**: Start plugin with health endpoint enabled and curl `/healthz`.

**Acceptance Scenarios**:

1. **Given** `FINFOCUS_PLUGIN_HEALTH_ENDPOINT=true`, **When** a GET request is made to `/healthz`, **Then** the server returns 200 OK.
2. **Given** `FINFOCUS_PLUGIN_HEALTH_ENDPOINT=false` (default), **When** a GET request is made to `/healthz`, **Then** the server returns 404 or does not expose the endpoint.

---

### Edge Cases

- **Invalid Max Age**: If `FINFOCUS_CORS_MAX_AGE` is not a valid integer, the system should log a warning and use the default (86400s).
- **Empty Origins**: If `FINFOCUS_CORS_ALLOWED_ORIGINS` is set but empty, no CORS headers should be generated.
- **Whitespace handling**: Comma-separated origins should be trimmed of whitespace (e.g., `a.com, b.com` -> `a.com`, `b.com`).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST parse `FINFOCUS_CORS_ALLOWED_ORIGINS` environment variable as a comma-separated list of origins.
- **FR-002**: System MUST configure the web server to return appropriate CORS headers for origins matching the parsed list.
- **FR-003**: System MUST log a warning if `FINFOCUS_CORS_ALLOWED_ORIGINS` is set to `*` (wildcard).
- **FR-004**: System MUST parse `FINFOCUS_CORS_ALLOW_CREDENTIALS` (case-insensitive "true"/"false") and enable credentials support if true.
- **FR-005**: System MUST log a fatal error and terminate at startup if `FINFOCUS_CORS_ALLOW_CREDENTIALS` is true while `FINFOCUS_CORS_ALLOWED_ORIGINS` is `*`.
- **FR-006**: System MUST parse `FINFOCUS_PLUGIN_HEALTH_ENDPOINT` (case-insensitive "true"/"false") and expose a `/healthz` endpoint returning HTTP 200 for GET requests if true. Other methods to `/healthz` MUST return 405 Method Not Allowed or 404.
- **FR-007**: System MUST parse `FINFOCUS_CORS_MAX_AGE` as an integer (seconds) and configure the CORS preflight max age.
- **FR-008**: System MUST fall back to a default max age of 86400 seconds if `FINFOCUS_CORS_MAX_AGE` is invalid or unset.
- **FR-009**: System MUST log the applied CORS configuration at startup (debug level) for verification.

### Key Entities

- **CORS Configuration**: The set of rules (origins, credentials, max age) that determine how the system responds to cross-origin requests.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can successfully make HTTP requests to the plugin from a browser application hosted on a specified allowed origin (integration test verified).
- **SC-002**: Kubernetes or other orchestrators can successfully probe `/healthz` when enabled.
- **SC-003**: Invalid configurations (e.g., credentials + wildcard) are handled safely without crashing or creating security holes (secure defaults).