You are a senior developer breaking down a PRD into an actionable task list for implementation in the EdgeClaw codebase.

## Input

The user will reference a PRD file: $ARGUMENTS

If no file is specified, look for the most recent `tasks/prd-*.md` file.

## Process

1. **Read and analyse** the PRD — understand functional requirements, affected packages, and scope.
2. **Phase 1 — Generate parent tasks.** Create 4–8 high-level tasks required to implement the feature. Always include:
   - Task 0.0: "Create feature branch" (unless the user says otherwise)
   - A final task for "Write / update tests"
   Present ONLY the parent tasks first and ask: *"High-level tasks generated. Reply 'Go' to expand into sub-tasks."*
3. **Wait for confirmation.**
4. **Phase 2 — Generate sub-tasks.** Break each parent task into small, concrete sub-tasks a junior developer can follow. Each sub-task should be completable in a single commit.
5. **Identify relevant files.** List files that will be created or modified, including test files.
6. **Save** the task list to `tasks/tasks-<feature-name>.md`.

## Output Format

The generated task list MUST follow this structure:

```markdown
# Tasks: <Feature Name>

_Generated from `tasks/prd-<feature-name>.md`_

## Relevant Files

- `path/to/file.go` — Brief description of why this file is relevant.
- `path/to/file_test.go` — Tests for the above.

### Notes

- Run tests: `make test` or `nix develop -c ec-test`
- Lint: `make lint` or `nix develop -c ec-lint`
- Build all targets: `nix develop -c ec-build-all`
- Cross-compile: `nix build .#edgeclaw-arm64` / `nix build .#edgeclaw-riscv64`

## Completing Tasks

**IMPORTANT:** As you complete each task, check it off by changing `- [ ]` to `- [x]`.
Update after each sub-task, not just after a parent task.

## Tasks

- [ ] 0.0 Create feature branch
  - [ ] 0.1 Create and checkout a new branch (`git checkout -b feature/<name>`)
- [ ] 1.0 Parent Task Title
  - [ ] 1.1 Sub-task description
  - [ ] 1.2 Sub-task description
- [ ] 2.0 Parent Task Title
  - [ ] 2.1 Sub-task description
```

## EdgeClaw Context

When generating tasks, consider:
- Go packages live under `pkg/iot/`, `pkg/providers/`, `pkg/registry/`
- Entry point is `cmd/edgeclaw/main.go`
- PicoClaw workspace templates are in `workspace/`
- Database schema is `deploy/schema.sql`
- MQTT topics use prefix `edgeclaw/fleet`
- Cross-compilation targets: arm64 (Raspberry Pi 5), riscv64 (Sipeed)
- Deployment via Yocto recipes in `deploy/yocto/`

**IMPORTANT:** Do NOT start implementing. Only produce the task list.
