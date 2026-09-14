# onCode Implementation Roadmap

## 1. Objective

The first success criterion is not “replicate all Cursor features.”

The first success criterion is:

> Build a safe, air-gapped, centrally orchestrated coding agent that can inspect a local project, propose a design, obtain approval, propose code changes, obtain diff approval, apply them locally, and validate through build/test.

---

# 2. Delivery Stages

## P0 — Prototype

Goal:

```text
Server ↔ Local Agent tool calling works.
```

Scope:

- protocol foundations
- Local Agent daemon
- file read
- apply prototype
- build prototype
- no IDE required

Exit:

- test client can call Local Agent reliably
- workspace boundary is enforced

---

## P1 — Technical MVP

Goal:

```text
Complete safe end-to-end coding workflow.
```

Scope:

- Java/Maven first
- VS Code first
- project search
- actual source read
- design
- design approval
- implementation proposal
- local actual diff
- code approval
- atomic apply
- Maven build/test
- basic fix loop
- basic audit
- workflow persistence/resume

Exit:

- core golden workflows pass repeatedly
- stale apply is prevented
- server restart can resume waiting workflows

---

## P2 — Developer Pilot

Add:

- Guide Retrieval
- Guide Ingestion/Admin
- multiple projects
- Git commit
- Node or Python support
- richer Project Intelligence
- basic enterprise auth/session
- developer pilot feedback

---

## P3 — Security Pilot

Add:

- Nexus-only dependency integration
- vulnerability DB
- license checks
- Security Agent
- stronger policy engine
- RBAC
- audit search
- secret detection

---

## P4 — Production

Add:

- runtime server HA
- DB HA/backup/restore
- shared artifact storage
- central monitoring/alerts
- rolling deployment
- agent version enforcement
- operational runbooks
- recovery drills

---

# 3. Implementation Phases

## Phase 0 — Protocol Foundation

Deliver:

- common IDs
- error model
- Protocol v1
- protobuf schemas
- JSON schemas where applicable
- contract tests
- protocol versioning

Do this first.

Executable task specification: [`../tasks/PHASE_00.md`](../tasks/PHASE_00.md)

---

## Phase 1 — Local Agent Core

Deliver:

- Go daemon
- server connector skeleton
- workspace registration
- workspace boundary
- canonical path validation
- symlink escape protection
- tool registry
- file read
- file search
- proposed change handling
- local diff
- atomic apply
- Git status/diff
- build/test adapters
- heartbeat
- basic local policy

This is the first implementation phase after protocol work.

Executable task specification: [`../tasks/PHASE_01.md`](../tasks/PHASE_01.md)

---

## Phase 2 — Server Workflow Core

Deliver:

- Spring Boot server
- PostgreSQL
- Work Item / Workflow state
- Context Service
- Tool Gateway
- Agent Runtime
- Inference Gateway
- basic Senior Developer
- Implementation Agent
- Review Agent
- approval state

Use mock/simple project context if necessary before Project Intelligence is complete.

Executable task specification: [`../tasks/PHASE_02.md`](../tasks/PHASE_02.md)

---

## Phase 3 — IDE Interaction

First IDE: VS Code unless organizational constraints require otherwise.

Deliver:

- chat
- progress
- question UI
- design review
- design approve/reject
- diff review
- code approve/reject
- build/test result
- reconnect

Executable task specification: [`../tasks/PHASE_03.md`](../tasks/PHASE_03.md)

---

## Phase 4 — Project Intelligence

Deliver:

- workspace scanner
- Tree-sitter integration
- symbol extraction
- import/relation extraction
- dependency metadata
- project index
- central catalog delta sync
- file/symbol search
- incremental watcher

Initial priority:

```text
relevant file discovery
>
symbol accuracy
>
summary quality
>
complete call graph
```

Executable task specification: [`../tasks/PHASE_04.md`](../tasks/PHASE_04.md)

---

## Phase 5 — End-to-End Implementation Loop

Close the real coding loop:

```text
request
→ project.search
→ source read
→ design
→ approval
→ implementation
→ proposed changes
→ local diff
→ approval
→ apply
→ build
→ test
```

Add bounded fix loops.

This phase defines the Technical MVP.

Executable task specification: [`../tasks/PHASE_05.md`](../tasks/PHASE_05.md)

---

## Phase 6 — Guide

Implement:

### 6A Ingestion

- PDF
- DOCX
- PPTX
- HWP/HWPX
- Markdown
- Docling
- LibreOffice fallback
- normalized markdown
- structured sections
- summary / keyword extraction
- curation console
- publish

### 6B Runtime Retrieval

- release pinning
- metadata filter
- keyword
- FTS
- trigram
- technology/category/version
- mandatory rules
- guide references

Executable task specification: [`../tasks/PHASE_06.md`](../tasks/PHASE_06.md)

---

## Phase 7 — Dependency / Security

Implement:

- current dependency analysis
- Nexus search
- Nexus resolve
- compatibility
- vulnerability precheck
- license check
- dependency decision
- approved build-file changes
- actual resolution
- dependency graph recheck
- critical/high block

Executable task specification: [`../tasks/PHASE_07.md`](../tasks/PHASE_07.md)

---

## Phase 8 — Authentication / Operations / Hardening

Implement:

- internal IdP integration
- session
- Local Agent registration
- project membership/RBAC
- audit
- structured logging
- metrics
- health checks
- runtime HA preparation
- backup/restore
- operational dashboards

Executable task specification: [`../tasks/PHASE_08.md`](../tasks/PHASE_08.md)

---

# 4. Critical Path

Do not lose focus on this path:

```text
Protocol
→ Local Agent File Tool
→ Server Tool Gateway
→ Workflow
→ Agent Runtime
→ IDE Approval
→ Local Actual Diff
→ Apply
→ Build/Test
```

Guide, Nexus, HA, and advanced multi-agent behavior must not delay this path.

---

# 5. Initial Agent Scope

P1 agents should be minimal:

```text
Senior Developer
Implementation
Review
```

Senior Developer may initially absorb:

- intent classification
- requirement analysis
- context planning
- design

Later split roles when evidence shows value.

---

# 6. First Supported Stack

Recommended:

```text
Java 17/21
Spring Boot
Maven
JUnit
```

Second:

```text
Node/npm
React/Vue
```

Third:

```text
Python
pytest
```

---

# 7. First Vertical Slices

## Slice 1

```text
Modify Greeter.java return value.
```

Validate:

- read file
- design
- approval
- propose content
- local diff
- approval
- apply
- test

## Slice 2

Add one method.

## Slice 3

Multi-file service change.

## Slice 4

Intentionally introduce/fix compile failure.

## Slice 5

Developer changes file between proposal and approval; apply must fail stale.

---

# 8. Mandatory P1 Tests

- workspace boundary
- symlink escape
- missing file
- file hash mismatch
- atomic multi-file apply
- design rejection
- code rejection
- approval reuse rejection
- expired approval
- stale workspace
- build failure classification
- test failure handling
- server restart in WAITING_APPROVAL
- Local Agent disconnect/reconnect
- duplicate side-effect call protection

---

# 9. Technical MVP Exit Criteria

Technical MVP is complete when all are true:

1. Java/Maven sample project supported.
2. VS Code supported.
3. Project search returns relevant files.
4. Actual source is read locally.
5. Design approval works.
6. Multi-file change proposal works.
7. Local actual diff is displayed.
8. Code approval works.
9. Apply is atomic.
10. Stale workspace is rejected.
11. Build/test run locally.
12. Build fix loop works.
13. Basic audit is persisted.
14. Waiting workflow survives server restart.

---

# 10. Things Not to Implement Early

Avoid these before P1 succeeds:

- large agent graph
- vector DB
- Kafka
- Redis without measured need
- Kubernetes as a development prerequisite
- advanced Git push/rebase/merge automation
- large-scale refactoring
- inline autocomplete
- hunk-level approval
- complete semantic call graph for all languages
- cross-repository implementation
