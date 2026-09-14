# Phase 01 Task — Local Agent Core

## 1. Goal

Implement the first production-shaped Local Agent core for onCode.

This phase is intentionally local-agent focused.

Do not implement the full server workflow yet.

Prerequisite: Phase 0 protocol (`PHASE_00.md`).

The Local Agent must safely expose local workspace tools through a protocol-compatible tool dispatcher and must enforce the workspace security boundary independently of the server.

---

# 2. Required Reference Documents

Before implementation, read:

## 2.1 Overview

1. `../overview/01_ARCHITECTURE.md` — onCode Architecture Overview
2. `../overview/02_IMPLEMENTATION_RULES.md` — onCode Mandatory Implementation Rules
3. `../overview/03_ADR_SUMMARY.md` — onCode ADR Summary
4. `../overview/04_IMPLEMENTATION_ROADMAP.md` — onCode Implementation Roadmap
5. `PHASE_00.md` — Protocol Foundation
6. this task document

## 2.2 Relevant SPECs

Phase 1 Local Agent Core에 해당하는 `docs/spec` 문서:

1. `../spec/SPEC-01 onCode Tool Protocol v1.md`  
   — onCode Tool Protocol v1 Specification  
   Tool request/response, error model, transport-independent protocol

2. `../spec/SPEC-04 onCode Local Agent Architecture v1.md`  
   — SPEC-04 onCode Local Agent Architecture v1  
   Local Agent 내부 구조, tool dispatcher, workspace, diff, apply, Git, build/test

3. `../spec/SPEC-08 onCode Security - Policy - Approval Model v1.md`  
   — SPEC-08 onCode Security / Policy / Approval Model v1  
   Workspace boundary, path/symlink 검증, apply approval. Phase 1부터 부분 적용

4. `../spec/SPEC-13 onCode Audit - Logging - Observability Architecture v1.md`  
   — SPEC-13 onCode Audit / Logging / Observability Architecture v1  
   Basic local audit/logging. P1 최소 기능만

5. `../spec/SPEC-15 onCode MVP Scope - Implementation Roadmap v1.md`  
   — SPEC-15 onCode MVP Scope / Implementation Roadmap v1  
   Phase 1 기능, tool 목록, 보안, 테스트, 완료 기준

Do not implement from later-phase SPECs in this task:

- SPEC-02 Workflow State Machine — Phase 2
- SPEC-03 Context Storage Data Model / ERD — Phase 2
- SPEC-05 Project Intelligence / PROJECT_INDEX Schema — Phase 4
- SPEC-06 Server Multi-Agent Architecture — Phase 2~5
- SPEC-07 IDE Extension Architecture — Phase 3
- SPEC-09 Guide Ingestion / Preprocessing / Curation — Phase 6
- SPEC-10 Runtime Guide Repository / Retrieval — Phase 6
- SPEC-11 Nexus / Dependency / Vulnerability Integration — Phase 7
- SPEC-12 Authentication / Session / Authorization — Phase 8
- SPEC-14 Deployment / HA / Operations Architecture — Production

## 2.3 Relevant ADRs

- ADR-002 Local Agent = Go
- ADR-003 Server ↔ Local Agent = gRPC bidirectional streaming
- ADR-011 Structured ProposedChanges + Local actual diff
- ADR-012 Local Project Intelligence responsibility — 책임 경계만. Tree-sitter indexing은 이 phase 범위 밖 (SPEC-05 / Phase 4)

---

# 3. In Scope

Implement:

```text
Local Agent process
Tool registry / dispatcher
Workspace registration
Workspace boundary enforcement
Canonical path handling
Symlink escape prevention
File read
Multi-file read
Basic file search
File hashing
Proposed change validation
Actual diff generation
Atomic apply
Git status
Git diff
Build adapter framework
Test adapter framework
Maven build
Maven test
Heartbeat
Capabilities
Structured error responses
Basic local audit/logging
```

---

# 4. Out of Scope

Do not implement in this phase:

```text
Full Central Server workflow
LLM integration
IDE extension
Guide repository
Nexus search
Security Agent
Git commit
Git push
Git merge
Git rebase
Arbitrary shell.execute
Tree-sitter project indexing
Multi-user auth
Redis
Kafka/message broker
HA
```

Do not expand scope unless explicitly instructed.

---

# 5. Recommended Module Structure

```text
local-agent/

├─ cmd/
│  └─ oncode-agent/
│
├─ internal/
│  ├─ agent/
│  ├─ config/
│  ├─ protocol/
│  ├─ tools/
│  │  ├─ system/
│  │  ├─ workspace/
│  │  ├─ git/
│  │  ├─ build/
│  │  └─ test/
│  ├─ workspace/
│  ├─ diff/
│  ├─ process/
│  ├─ policy/
│  ├─ security/
│  ├─ audit/
│  └─ transport/
│
├─ pkg/
│
└─ tests/
```

Adapt if the repository already has a stronger established structure.

Do not reorganize unrelated code without need.

---

# 6. Core Domain Types

At minimum, model:

```text
AgentCapabilities
Workspace
WorkspaceRevision
ToolRequest
ToolResponse
ToolError
ProposedChangeSet
FileChange
ActualDiff
ApprovalBinding
ExecutionResult
BuildResult
TestResult
```

Use protocol-defined identifiers when the shared protocol module is available.

---

# 7. Workspace Registration

A workspace has:

```text
workspace_id
project_id optional at this phase
root_path
registered_at
current_revision
```

The Local Agent internally knows the absolute root.

Remote/tool APIs should prefer workspace-relative paths.

Example:

```text
src/main/java/com/acme/AuthService.java
```

not:

```text
C:\Users\developer\project\src\...
```

---

# 8. Path Security Requirements

Every file tool must:

1. Receive a registered `workspace_id`.
2. Resolve the workspace root.
3. Reject empty/invalid paths where inappropriate.
4. Resolve the requested path canonically.
5. Ensure the final resolved path remains inside the workspace root.
6. Evaluate symlinks safely.
7. Reject escape attempts.

Must reject examples such as:

```text
../secret.txt
../../etc/passwd
C:\Windows\System32\...
/etc/passwd
workspace/symlink -> outside-root
```

Use explicit error codes.

---

# 9. File Hashing

Use SHA-256 for file content hashes.

Return hashes in a consistent form such as:

```text
sha256:<hex>
```

MODIFY / DELETE / RENAME proposals require a base hash.

---

# 10. Required Tools

## 10.1 `system.ping`

Request:

```json
{}
```

Response includes:

```text
agent_id
version
protocol_version
timestamp
```

---

## 10.2 `system.get_capabilities`

Return:

- OS
- arch
- agent version
- supported protocol version
- supported tools
- detected build tools where feasible

Example capabilities:

```text
git
maven
java
```

Do not overbuild environment detection in this phase.

---

## 10.3 `workspace.read_file`

Input:

```text
workspace_id
path
```

Output:

```text
path
content
hash
size
```

Requirements:

- workspace-relative path
- text file only for MVP
- explicit encoding handling
- configurable maximum file size
- no outside-root read

---

## 10.4 `workspace.read_files`

Input:

```text
workspace_id
paths[]
```

Return per-file success/failure.

A single missing file does not necessarily need to invalidate every other read, but response behavior must be deterministic and documented.

---

## 10.5 `workspace.search`

MVP search is intentionally basic.

Support:

- file name search
- path substring search
- text literal search

Do not implement semantic search or Tree-sitter here.

Important exclusions should be configurable:

```text
.git
target
node_modules
dist
build
binary files
```

---

## 10.6 `workspace.propose_changes`

This tool does **not** mutate files.

Input:

```text
change_set_id
workspace_id
base_workspace_revision
changes[]
```

File operations:

```text
CREATE
MODIFY
DELETE
RENAME
```

Responsibilities:

- validate change structure
- validate paths
- validate base hashes
- materialize proposed target content
- compare to current workspace
- generate actual unified diff
- generate a deterministic diff hash
- return proposal metadata

Do not apply changes.

---

## 10.7 `workspace.apply_changes`

Requirements:

Input includes at minimum:

```text
change_set_id
workspace_id
approval_id
approved_diff_hash
```

This phase may use a mock/local approval validator interface, but the boundary must exist.

Apply algorithm:

```text
Load approved proposal
→ Validate workspace
→ Re-read all current files
→ Revalidate base hashes
→ Recompute/validate approved diff identity
→ Stage changes
→ Apply atomically
→ Return updated hashes/revision
```

If any precondition fails:

```text
apply none
```

Do not partially apply.

---

# 11. Atomic Apply Strategy

Implement an explicit transaction-like local strategy.

Recommended approach:

1. Validate all operations.
2. Prepare temporary files.
3. Prepare backup metadata.
4. Apply renames/writes/deletes in controlled order.
5. On failure, roll back.
6. Only then mark the change set applied.

Tests must simulate failure partway through a multi-file change.

---

# 12. Actual Diff

The Local Agent owns the user-facing diff.

Use a stable unified-diff representation for transport/UI.

The diff is generated from:

```text
current local source
vs
proposed final content
```

Do not accept a model-generated diff as authoritative.

---

# 13. Git Tools

## `git.status`

Return structured:

```text
branch
head
changed files
untracked files
staged files
```

## `git.diff`

Support workspace-relative/project diff.

No commit, push, merge, rebase in this phase.

---

# 14. Build/Test Adapter

Define interfaces first.

Example conceptual API:

```text
BuildAdapter.Detect(workspace)
BuildAdapter.Run(context, request)

TestAdapter.Detect(workspace)
TestAdapter.Run(context, request)
```

First implementation:

```text
Maven
```

Do not design the interface so narrowly that Gradle/npm/pytest cannot be added later.

---

# 15. Maven Build

Implement safe explicit Maven execution.

Candidate behavior:

```text
mvn -B -DskipTests compile
```

or repository-defined wrapper:

```text
./mvnw
mvnw.cmd
```

Prefer project wrapper when present.

Do not execute arbitrary user-provided shell strings.

Return structured:

```text
status
exit_code
duration_ms
stdout_summary
stderr_summary
diagnostics[]
full_log_ref optional
```

---

# 16. Maven Test

Run tests through the Maven adapter.

Return:

```text
status
exit_code
duration_ms
total
passed
failed
skipped
failures[]
```

Parsing can start simple but must preserve raw output/log reference for future parsers.

---

# 17. Process Execution Security

All processes must:

- run inside the workspace or explicit approved subdirectory
- use argument arrays, not concatenated shell strings
- support timeout
- support cancellation
- capture exit code
- bound log size
- kill child process tree when needed

No generic arbitrary shell endpoint in Phase 1.

---

# 18. Structured Error Model

Use stable error codes.

At minimum:

```text
WORKSPACE_NOT_FOUND
INVALID_PATH
PATH_OUTSIDE_WORKSPACE
SYMLINK_ESCAPE
FILE_NOT_FOUND
FILE_TOO_LARGE
FILE_HASH_MISMATCH
WORKSPACE_FILE_CHANGED
INVALID_CHANGE_SET
CHANGE_SET_NOT_FOUND
APPROVAL_REQUIRED
APPROVAL_INVALID
DIFF_MISMATCH
APPLY_FAILED
BUILD_TOOL_NOT_FOUND
BUILD_FAILED
TEST_FAILED
PROCESS_TIMEOUT
PROCESS_CANCELLED
GIT_NOT_AVAILABLE
INTERNAL_ERROR
```

Do not expose raw internal stack traces as protocol errors.

Internal logs may retain sanitized diagnostics.

---

# 19. Workspace Revision

Implement a simple monotonic local workspace revision.

Revision increments when:

- Local Agent successfully applies a Change Set
- file watcher support later detects external changes

For Phase 1, at minimum increment on agent apply.

The design must allow file-watcher-driven increments later.

---

# 20. Persistence

Persist minimal Local Agent state outside the project workspace.

Persist:

```text
agent identity
registered workspaces
known change-set state
execution status needed for reconciliation
```

Do not put daemon logs/state under the project source tree except explicit `.codegen` project metadata where separately defined.

---

# 21. Logging

Use structured local logging.

Include:

```text
request/call id
workspace id
tool name
status
duration
error code
```

Do not log:

```text
full file content
credentials
tokens
private keys
```

---

# 22. Heartbeat / Transport Skeleton

Implement a transport abstraction compatible with future gRPC bidirectional streaming.

Phase 1 may use an integration test harness before the real server exists.

The code structure must support:

```text
connect
register
heartbeat
receive ToolRequest
send ToolResponse
reconnect
```

Do not tightly couple tool handlers to gRPC-generated types.

Use domain types plus protocol adapters.

---

# 23. Required Unit Tests

At minimum:

### Path security

- normal path
- nested path
- `../`
- absolute path
- symlink inside root
- symlink escaping root

### File tools

- read valid text
- missing file
- file hash
- multi-read mixed result
- configured max size

### Change proposal

- CREATE
- MODIFY correct hash
- MODIFY wrong hash
- DELETE
- RENAME
- invalid target path
- duplicate target
- conflicting operations

### Apply

- successful single file
- successful multi-file
- stale hash before apply
- diff hash mismatch
- invalid approval
- rollback on mid-apply failure
- apply same change set twice

### Process

- build success
- build fail
- timeout
- cancellation

---

# 24. Required Integration Tests

Use temporary/sample workspaces.

Provide at least:

```text
sample-java-maven
```

Tests:

1. read Greeter.java
2. propose modified Greeter.java
3. verify actual diff
4. approve through test approval adapter
5. apply
6. run Maven compile/test
7. verify resulting hash/revision

Also test:

```text
proposal
→ external file edit
→ apply
→ WORKSPACE_FILE_CHANGED
```

---

# 25. Definition of Done

Phase 1 is complete only when:

1. Go Local Agent builds on target developer OS.
2. Workspace boundary tests pass.
3. File reads are deterministic and hashed.
4. `workspace.propose_changes` never mutates files.
5. Local Agent generates the actual diff.
6. `workspace.apply_changes` requires approval validation.
7. Stale files prevent apply.
8. Multi-file apply is atomic.
9. Git status/diff work.
10. Maven build/test adapters work on sample project.
11. Tool errors are structured.
12. Tool handlers are decoupled from transport.
13. Integration tests pass.
14. No arbitrary shell endpoint exists.
15. No public network dependency is introduced.

---

# 26. Implementation Instructions for Coding Agents

Before changing code:

1. Inspect the repository.
2. Identify existing conventions.
3. Produce an implementation plan.
4. List files/modules to create or modify.
5. Identify any conflict with architecture rules.
6. Do not redesign unrelated components.

During implementation:

- make small cohesive commits/changes
- add tests with each capability
- prefer explicit code over speculative abstraction
- keep future adapters possible, but avoid premature framework-building

After implementation:

1. Run unit tests.
2. Run integration tests.
3. Run the sample Maven project.
4. Report architecture deviations.
5. Report remaining Phase 1 gaps.
6. Do not proceed to Phase 2 unless explicitly requested.
