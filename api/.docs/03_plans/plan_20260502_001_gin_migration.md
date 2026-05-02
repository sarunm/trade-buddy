# Plan - Task 001: Gin Migration (T1.2b)

- Task ID: 001
- Created: 2026-05-02 14:10
- Status: Planning
- Priority: Critical

## User Story / Objective
As a developer, I want to use the Gin framework for the HTTP layer so that I can leverage its performance and built-in features for API development.

## Acceptance Criteria
- [ ] Gin dependency added to `go.mod`.
- [ ] `NewRouter` in `internal/http/router.go` returns `*gin.Engine`.
- [ ] `health` handler in `internal/http/health.go` uses `gin.Context`.
- [ ] `internal/http/json.go` deleted if it only contains `writeJSON`.
- [ ] Security middleware in Gin sets `X-Content-Type-Options: nosniff`.
- [ ] `go build ./...` passes.

## Technical Design & Architecture Review

**Chosen Approach:**
I will use `github.com/gin-gonic/gin` to replace `net/http`'s `ServeMux`. Gin's `*gin.Engine` implements the `http.Handler` interface, so it will be a drop-in replacement in `main.go`.

**Implementation Steps:**
1.  Run `go get github.com/gin-gonic/gin`.
2.  Update `internal/http/router.go`:
    - Change imports.
    - Change `NewRouter` return type.
    - Initialize Gin engine.
    - Use Gin middleware for common headers.
    - Define routes using Gin's router.
3.  Update `internal/http/health.go`:
    - Change imports.
    - Update `health` handler signature and body.
4.  Check `internal/http/json.go`:
    - Delete if redundant.
5.  Verify `cmd/server/main.go` still works (it should, as `*gin.Engine` is an `http.Handler`).
6.  Run `go build ./...` and `go mod tidy`.

## Risk Assessment
- **Middleware Differences:** Gin's middleware pattern is different from stdlib. I'll use Gin's built-in ways or write a Gin-compatible middleware.

## Dependencies & Blockers
- Internet access for `go get`.
