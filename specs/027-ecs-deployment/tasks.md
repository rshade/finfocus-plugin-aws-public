# Tasks: Add Amazon ECS Deployment Example

**Feature Branch**: `027-ecs-deployment`
**Feature Spec**: `specs/027-ecs-deployment/spec.md`

## Phase 1: Setup
*Goal: Initialize documentation structure.*

- [x] T001 Create new file `docs/ecs-deployment.md`

## Phase 2: Foundational
*Goal: Establish document structure and introduction.*

- [x] T002 Add title, "Introduction", and "Architecture" overview to `docs/ecs-deployment.md`

## Phase 3: US1 - Deploy Plugin to ECS (P1)
*Goal: Provide a copy-paste ready ECS Task Definition and sizing guidance.*
*Test: User can register the provided Task Definition JSON successfully.*

- [x] T003 [US1] Add "ECS Task Definition" section with complete JSON example (Fargate, 12 ports) to `docs/ecs-deployment.md`
- [x] T004 [US1] Add "Resource Sizing" section explaining 2 vCPU / 4 GB RAM rationale to `docs/ecs-deployment.md`

## Phase 4: US2 - Service Networking (P1)
*Goal: Explain how to expose the multi-port service using Cloud Map.*
*Test: User understands why Service Discovery is preferred over ALB/NLB.*

- [x] T005 [P] [US2] Add "Networking Strategy" section comparing Service Discovery vs Load Balancers to `docs/ecs-deployment.md`
- [x] T006 [P] [US2] Add "Accessing the Plugin" section with internal DNS resolution examples (e.g., `plugin.local:8001`) to `docs/ecs-deployment.md`

## Phase 5: US3 - Environment Variables (P2)
*Goal: Document configuration options for the plugin container.*
*Test: User can find all supported variables and their default values.*

- [x] T007 [US3] Add "Environment Variables" section with table (WEB_ENABLED, LOG_LEVEL, CORS) to `docs/ecs-deployment.md`

## Phase 6: US4 - Prerequisites & Validation (P2)
*Goal: Ensure users have the necessary AWS environment and can troubleshoot issues.*
*Test: User can follow the checklist to prepare their VPC and IAM roles.*

- [x] T008 [P] [US4] Add "Prerequisites" checklist (VPC, Subnets, Security Groups, IAM Roles) to `docs/ecs-deployment.md`
- [x] T009 [P] [US4] Add "Troubleshooting" section covering common failures (Service Discovery, IAM) to `docs/ecs-deployment.md`
- [x] T010 [P] [US4] Add "Terraform Example" section with HCL snippet for Task Def & Service to `docs/ecs-deployment.md`

## Phase 7: Pulumi Example (New Requirement)
*Goal: Provide a ready-to-use Pulumi YAML program for deployment.*

- [x] T011 Create directory `examples/pulumi-ecs`
- [x] T012 Create `examples/pulumi-ecs/Pulumi.yaml` with the Pulumi YAML program (ECS Fargate + Cloud Map)
- [x] T013 Add "Pulumi YAML Example" section to `docs/ecs-deployment.md` referencing the example code

## Phase 8: Polish
*Goal: Integrate new documentation into the project.*

- [x] T015 Update `README.md` to link to `docs/ecs-deployment.md` in the "Documentation" or "Deployment" section

## Dependencies
1. **Setup & Foundation**: T001, T002 MUST complete first.
2. **US1 & US2 (P1)**: T003-T006 can run in parallel after Foundation.
3. **US3 & US4 (P2)**: T007-T010 can run in parallel with US1/US2 (content is independent).
4. **Polish**: T011 requires `docs/ecs-deployment.md` to exist.

## Parallel Execution Examples
- **Documentation Writing**: T005 (Networking), T008 (Prereqs), and T010 (Terraform) can be written simultaneously by different writers.
- **Review**: T003 (Task Def) and T007 (Env Vars) can be reviewed independently.

## Implementation Strategy
- **MVP**: Complete Phases 1, 2, 3, and 4 (US1 & US2) to allow users to deploy.
- **Full Scope**: Add Phases 5 and 6 (US3 & US4) for complete operational guidance.
