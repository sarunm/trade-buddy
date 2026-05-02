# Plan - Task 002: Dockerization (T1.3)

- Task ID: 002
- Created: 2026-05-02 14:35
- Status: Planning
- Priority: High

## User Story / Objective
As a developer, I want to containerize the API so that it can be easily deployed and scaled.

## Acceptance Criteria
- [ ] `Dockerfile` exists in `api/`.
- [ ] `Dockerfile` uses multi-stage build (`golang:1.23-alpine` for build, `alpine:latest` for runtime).
- [ ] `docker build -t trade-buddy-api .` succeeds.

## Technical Design & Architecture Review

**Chosen Approach:**
Multi-stage Docker build to keep the final image size small.
- Stage 1 (builder): Compiles the Go binary.
- Stage 2 (runtime): Runs the compiled binary.

**Implementation Steps:**
1.  Create `Dockerfile` with the content provided in the request.
2.  Run `docker build -t trade-buddy-api .` in the `api/` directory.

## Risk Assessment
- **Build Errors:** Ensure `go.sum` is present (it should be after `go mod tidy`).
- **Binary Path:** Verify the binary is built to `/app/server` as expected in the Dockerfile.

## Dependencies & Blockers
- Docker must be installed and running on the host machine.
