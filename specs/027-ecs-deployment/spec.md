# Feature Specification: Add Amazon ECS Deployment Example

**Feature Branch**: `027-ecs-deployment`
**Created**: 2026-01-16
**Status**: Draft
**Input**: Create a comprehensive documentation guide and reference configuration for deploying the multi-region Docker image to Amazon ECS (Fargate).

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.
  
  Assign priorities (P1, P2, P3, etc.) to each story, where P1 is the most critical.
  Think of each story as a standalone slice of functionality that can be:
  - Developed independently
  - Tested independently
  - Deployed independently
  - Demonstrated to users independently
-->

### User Story 1 - Deploy Plugin to ECS via Task Definition Reference (Priority: P1)

An infrastructure engineer needs to deploy the multi-region finfocus-plugin-aws-public Docker image to AWS ECS Fargate without manually designing a task definition. They want a concrete, copy-paste-ready example with sensible defaults for CPU, memory, and port configuration.

**Why this priority**: This is the core user need - enabling anyone to deploy the plugin to ECS with minimal AWS expertise. Without this, users must reverse-engineer configuration from scratch.

**Independent Test**: Documentation provides a complete task definition that can be deployed as-is without modification and successfully exposes all 12 regional ports for internal service discovery.

**Acceptance Scenarios**:

1. **Given** a user has the ECS deployment documentation, **When** they copy the provided task definition JSON, **Then** they can create an ECS task definition with a single `aws ecs register-task-definition` command
2. **Given** an ECS task is running with the provided configuration, **When** they query internal service discovery endpoints, **Then** all 12 regional ports (8001-8012) are reachable from other VPC services
3. **Given** the task definition example, **When** they review the CPU/memory settings, **Then** they find documented resource sizing recommendations with rationale

---

### User Story 2 - Understand Service Networking & Access Patterns (Priority: P1)

A DevOps engineer needs to understand how to make the multi-port plugin accessible to other services in their VPC. They want clear guidance on networking strategies (ALB vs NLB vs Service Discovery) and which approach to use for their use case.

**Why this priority**: Understanding networking architecture is essential for production deployments. Poor choices here impact reliability and operational complexity. This directly affects whether the deployment succeeds.

**Independent Test**: Documentation explains service discovery approach, ALB/NLB trade-offs, and provides configuration examples for the recommended pattern (Cloud Map service discovery).

**Acceptance Scenarios**:

1. **Given** documentation on ECS networking strategies, **When** a user reads about ALB limitations, **Then** they understand why ALBs have listener limits and why NLB or Service Discovery is recommended for multi-port workloads
2. **Given** the recommended Cloud Map service discovery approach, **When** another service queries the discovered endpoint, **Then** it can access all 12 regional ports without additional load balancer configuration
3. **Given** service discovery documentation, **When** a user needs to add a new regional port, **Then** they can do so without modifying load balancer listener rules

---

### User Story 3 - Configure Plugin Environment Variables in ECS Task (Priority: P2)

An operator wants to customize the plugin's behavior via environment variables (CORS origins, logging level, health endpoint) when deploying to ECS. They need to know which variables are supported and where to set them in the task definition.

**Why this priority**: While important for production deployments, this is secondary to basic deployment. Users can start with defaults and customize later as needs evolve.

**Independent Test**: Documentation lists all supported environment variables with examples and shows where/how to configure them in the ECS task definition.

**Acceptance Scenarios**:

1. **Given** the task definition documentation, **When** they review the environment section, **Then** they see examples of key variables (FINFOCUS_CORS_ALLOWED_ORIGINS, FINFOCUS_PLUGIN_WEB_ENABLED) with explanation
2. **Given** a need to enable web serving with CORS, **When** they apply the documented environment variable pattern, **Then** the plugin starts with web serving enabled and appropriate CORS headers

---

### User Story 4 - Prerequisites and Network Setup Validation (Priority: P2)

A new user wants to verify they have everything needed before attempting deployment. They need clear prerequisites including VPC/subnet requirements and guidance on validating their setup.

**Why this priority**: This helps users avoid common setup mistakes and understand environment requirements. Valuable for education but not blocking basic deployment.

**Independent Test**: Documentation provides a prerequisites checklist that users can complete before deployment (VPC exists, subnets in multiple AZs, IAM roles configured).

**Acceptance Scenarios**:

1. **Given** the prerequisites section, **When** a user reviews the checklist, **Then** they understand required AWS resources (VPC, subnets, IAM execution role, task role)
2. **Given** documentation on security groups and networking, **When** they configure inbound rules, **Then** they can reach the plugin ports from other VPC services

---

### Edge Cases

- What happens when using a single-AZ setup (documentation should note this is not recommended for production)?
- How should users handle port conflicts if multiple plugin versions need to run simultaneously (guidance on port mapping strategy)?
- What happens if the ECS cluster doesn't have sufficient capacity (CPU/memory) for the configured task (user must check cluster capacity before launching)?
- How do users troubleshoot if service discovery endpoints aren't resolving (documentation includes debugging steps)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Documentation MUST provide a complete ECS task definition example (JSON format) with the Docker image, CPU/memory, and all 12 port mappings
- **FR-002**: Documentation MUST explain the service networking strategy with recommendations for ALB vs NLB vs Service Discovery and rationale for each approach
- **FR-003**: Documentation MUST include a working Cloud Map (Service Discovery) configuration example showing how other services discover and reach the plugin ports
- **FR-004**: Documentation MUST document all supported environment variables (FINFOCUS_CORS_ALLOWED_ORIGINS, FINFOCUS_PLUGIN_WEB_ENABLED, FINFOCUS_LOG_LEVEL, etc.) with examples
- **FR-005**: Documentation MUST include a prerequisites section with VPC, subnet, security group, and IAM role requirements
- **FR-006**: Documentation MUST provide an (optional) Terraform example showing IaC-based deployment for users preferring Infrastructure as Code
- **FR-007**: Documentation MUST include troubleshooting guidance for common issues (service discovery not resolving, ports not accessible, container failing to start)
- **FR-008**: Documentation MUST specify recommended resource sizing (CPU, memory) with sizing rationale based on expected load

### Key Entities

- **ECS Task Definition**: Defines container configuration, resource allocation, port mappings, environment variables, and logging
- **ECS Service**: Long-running managed service that maintains desired task count and integrates with load balancers/service discovery
- **Cloud Map Namespace**: Private namespace for internal service discovery within the VPC
- **Security Group**: Controls inbound/outbound network traffic for the ECS tasks
- **IAM Execution Role**: Allows ECS agent to pull the Docker image and write logs
- **IAM Task Role**: (Optional) Allows the plugin container to call AWS APIs if needed for future features

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Documentation is complete with working task definition example that can be deployed with no modifications and successfully exposes all 12 ports
- **SC-002**: A user following the documentation can deploy the plugin to ECS and access it via service discovery within 15 minutes (excluding AWS API calls)
- **SC-003**: Documentation addresses all three deployment approaches (ALB, NLB, Service Discovery) with clear guidance on trade-offs
- **SC-004**: 90% of deployment issues can be resolved using the troubleshooting guide without external support
- **SC-005**: Documentation is accessible and understandable to operators with intermediate AWS knowledge (not requiring ECS/Fargate expertise)
- **SC-006**: All supported environment variables are documented with at least one example of usage
- **SC-007**: Documentation includes multiple formats (Markdown, JSON examples, optional Terraform) to accommodate different user preferences

## Assumptions

- Users have basic AWS knowledge and understand VPC, subnets, and security groups
- ECS is deployed with Fargate launch type (not EC2)
- The multi-region Docker image from #244 exposes ports 8001-8012 for regions
- Service Discovery (Cloud Map) is the recommended approach for internal VPC access (simpler than ALB/NLB listener management)
- The plugin binary is built with real pricing data for intended deployment region
- Users will use the image from `ghcr.io/rshade/finfocus-plugin-aws-public:latest` (public registry)
- No external database or persistent storage is needed (stateless plugin)
- Logging can be sent to CloudWatch Logs via awslogs driver (standard ECS pattern)
