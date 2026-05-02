# Orchestrator Rules — Multi-Agent Coordinator

Claude acts as the orchestrator. Agents (Gemini, Codex, Opencode) do the work.
Claude plans, assigns, monitors, reviews, and decides next steps.

---

## Agent Roster

| Agent    | Strength                                              | CLI                                      |
|----------|-------------------------------------------------------|------------------------------------------|
| Gemini   | Large context — port Python→Go, analysis, research    | `gemini -p "$(cat file)" --yolo`         |
| Codex    | Precise edits — Go/TS implementation, build+test loop | `codex exec --full-auto -C <dir> "..."`  |
| Opencode | Frontend — React, Next.js, Tailwind, scaffolding      | `opencode run "..."`                     |

Auto-assign by task ID:
- T3.2, T4.1–T4.5 → Gemini (port Python analysis)
- T5.x → Opencode (UI/React)
- Everything else → Codex

---

## Orchestration Flow

### 1. Plan
- Read `.claude/active.md` — find all `pending` tasks where deps are `done`
- Group tasks that can run in parallel (no shared file writes)
- Decide which can safely run concurrently vs must be sequential

### 2. Dispatch (parallel)
For each task in the ready group:
1. Build prompt from `docs/plan.md` task spec (Goal + Read first + Create + Test + Done when)
2. Prepend shared context:
   - Working directory: `/Users/nick/2_SideProjects/trade-buddy`
   - AGENTS.md contents (gist)
   - Relevant topics/ notes (if any)
3. Write prompt to `/tmp/tb_prompt_<TID>_<timestamp>.md`
4. Spawn Task tool with sub-agent instructions (see Sub-Agent Contract below)
5. Update `active.md`: status → `dispatched`, log entry added

### 3. Monitor
- Call TaskGet on all running task IDs
- Report progress to user every ~30s
- Do not re-dispatch until current batch completes

### 4. Review
When a task finishes (Task tool completes):
1. Read `.claude/tasks/<TID>-result.md`
2. Check: exit code, files exist, verification command output
3. Spot-check key logic (read 20–40 lines of created file)
4. Decision:
   - **Pass** → update `active.md` status → `done`, mark `[x]` in `docs/plan.md`, promote next deps to ready queue
   - **Retry same agent** → re-dispatch with additional context (error + what was wrong)
   - **Reassign** → change agent, note reason in dispatch log
   - **Research needed** → dispatch Gemini research task (see Research Flow)

### 5. Research Flow
When a task is blocked on knowledge (not on a dependency):
1. Create research prompt: "Read [files]. Answer: [specific question]. Write findings to .claude/topics/<topic>.md"
2. Dispatch to Gemini via Task tool
3. Mark original task status → `research`
4. When research done → re-dispatch original task with topics file reference added to prompt

---

## Sub-Agent Contract

When Claude spawns a Task for an agent, the Task prompt must include:

```
ROLE: You are a sub-agent runner. Run the external CLI agent and capture results.

TASK ID: <TID>
WORKING DIR: /Users/nick/2_SideProjects/trade-buddy
PROMPT FILE: /tmp/tb_prompt_<TID>_<ts>.md
AGENT: <gemini|codex|opencode>
RESULT FILE: /Users/nick/2_SideProjects/trade-buddy/.claude/tasks/<TID>-result.md

STEPS:
1. Run the agent:
   - Gemini: gemini -p "$(cat /tmp/tb_prompt_<TID>_<ts>.md)" --yolo > /tmp/tb_out_<TID>.log 2>&1
   - Codex:  codex exec --full-auto -C /Users/nick/2_SideProjects/trade-buddy "$(cat /tmp/tb_prompt_<TID>_<ts>.md)" < /dev/null > /tmp/tb_out_<TID>.log 2>&1
   - Opencode: opencode run "$(cat /tmp/tb_prompt_<TID>_<ts>.md)" < /dev/null > /tmp/tb_out_<TID>.log 2>&1

2. Wait for exit. Then write RESULT FILE:
   ## <TID> Result
   Agent: <name>
   Exit: <code>
   Files created/modified:
   - <list from checking filesystem>
   Verification: <run the test command from the task spec, capture output>
   Log (last 50 lines):
   <tail -50 /tmp/tb_out_<TID>.log>

3. Done — orchestrator will read the result file.
```

---

## Parallel Safety Rules

Tasks in the same parallel batch must NOT:
- Write to the same file (e.g., two tasks updating `router.go`)
- Depend on each other's output
- Both require DB running (only one DB test at a time)

When in doubt, run sequentially.

---

## Review Checklist by Type

**Go handler (Codex)**:
- `go build ./...` passes
- Route registered in router.go
- Response shape matches spec in migration plan

**Go port from Python (Gemini)**:
- Build passes
- Test returns ≥1 result
- Key logic comment explains mapping from Python

**React component (Opencode)**:
- `npm run build` passes (no TS errors)
- Component accepts correct props
- Dark theme compatible

**SQL migration (Codex)**:
- All tables from spec present
- Indexes defined
- Down migration exists

---

## Cleanup

After each task batch:
- Delete `/tmp/tb_prompt_*.md` and `/tmp/tb_out_*.log`
- Remove Gemini planning artifacts (`.docs/` folders if created)
- Archive task result files after phase gate passes: move `.claude/tasks/*.md` → `.claude/sessions/<date>-phase<N>/`
