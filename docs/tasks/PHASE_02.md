# Phase 02 Task — Server Workflow Core

## 1. Goal

Implement the central server workflow core after Local Agent tools exist.

This phase is server-orchestrator focused.

Do not implement IDE UI, Guide, Nexus, HA, or full Project Intelligence.

The server must persist workflow state, drive a bounded agent set, call Local Agent tools through a Tool Gateway, and stop at design → approval → implementation internally.

Use mock or simple project context if Project Intelligence is not yet complete.

Prerequisite: Phase 0 protocol and Phase 1 Local Agent Core.

---

# 2. Required Reference Documents

Before implementation, read:

## 2.1 Overview

1. `../overview/01_ARCHITECTURE.md` — onCode Architecture Overview
2. `../overview/02_IMPLEMENTATION_RULES.md` — onCode Mandatory Implementation Rules
3. `../overview/03_ADR_SUMMARY.md` — onCode ADR Summary
4. `../overview/04_IMPLEMENTATION_ROADMAP.md` — onCode Implementation Roadmap
5. `PHASE_01.md` — Local Agent capabilities this server will call
6. this task document

## 2.2 Relevant SPECs

Phase 2 Server Workflow Core에 해당하는 `docs/spec` 문서:

1. `../spec/SPEC-01 onCode Tool Protocol v1.md`  
   — onCode Tool Protocol v1 Specification  
   Server ↔ Local Agent ToolRequest/ToolResponse

2. `../spec/SPEC-02 onCode Workflow State Machine v1.md`  
   — SPEC-02 onCode Workflow State Machine v1  
   상태, 전이, 승인 대기. 이 phase의 핵심

3. `../spec/SPEC-03 onCode Context Storage Data Model - ERD v1.md`  
   — SPEC-03 onCode Context Storage Data Model / ERD v1  
   Work Item, Workflow, Approval, Change Set persistence

4. `../spec/SPEC-06 onCode Server Multi-Agent Architecture v1.md`  
   — SPEC-06 onCode Server Multi-Agent Architecture v1  
   Senior Developer / Implementation / Review. 최소 구성만

5. `../spec/SPEC-08 onCode Security - Policy - Approval Model v1.md`  
   — SPEC-08 onCode Security / Policy / Approval Model v1  
   Design/code approval state. Agent는 apply를 호출하지 않음

6. `../spec/SPEC-15 onCode MVP Scope - Implementation Roadmap v1.md`  
   — SPEC-15 onCode MVP Scope / Implementation Roadmap v1  
   Phase 2 구성, workflow, context entity, 완료 기준

Do not implement from later-phase SPECs in this task:

- SPEC-05 Project Intelligence / PROJECT_INDEX Schema — Phase 4 (mock context 허용)
- SPEC-07 IDE Extension Architecture — Phase 3
- SPEC-09 / SPEC-10 Guide — Phase 6
- SPEC-11 Nexus / Dependency / Vulnerability — Phase 7
- SPEC-12 Authentication / Session / Authorization — Phase 8 (Session 기본만)
- SPEC-14 Deployment / HA / Operations — Production

## 2.3 Relevant ADRs

- ADR-001 Central Server = Java + Spring Boot, Modular Monolith
- ADR-003 Server ↔ Local Agent = gRPC bidirectional streaming
- ADR-005 PostgreSQL-only for MVP/Pilot; PostgreSQL is workflow SSOT
- ADR-007 Agent Runtime → Inference Gateway → vLLM
- ADR-010 PostgreSQL persistent state machine + workflow lease
- ADR-011 Structured ProposedChanges + Local actual diff; agents must not apply

---

# 3. In Scope

Implement:

```text
Spring Boot server
PostgreSQL
Work Item / Workflow state
Context Service
Tool Gateway
Agent Runtime
Inference Gateway
basic Senior Developer
Implementation Agent
Review Agent
approval state
context.get / context.put
workflow persistence / resume
```

---

# 4. Out of Scope

Do not implement in this phase:

```text
VS Code / IDE extension
Tree-sitter project indexing
Guide ingestion / retrieval
Nexus / vulnerability DB
Security Agent
RBAC / internal IdP
Redis
Kafka / message broker
HA
inline autocomplete
hunk-level approval
large agent graph
```

Do not expand scope unless explicitly instructed.

---

# 5. Recommended Module Structure

```text
server/

├─ api/
├─ workflow/
├─ context/
├─ tool-gateway/
├─ agent-runtime/
├─ inference-gateway/
├─ approval/
└─ persistence/
```

Keep a modular monolith. Do not split microservices in this phase.

---

# 6. Minimum Agents

P1 agents must stay minimal:

```text
Senior Developer
Implementation
Review
```

Senior Developer may initially absorb:

```text
intent classification
requirement analysis
context planning
design
```

Requirement Agent and Security/Dependency Agent are out of scope.

Agents MUST NOT call `workspace.apply_changes`.

Agents exchange `context_refs`. They MUST NOT pass large payloads to one another.

---

# 7. Workflow States

Implement this first workflow:

```text
RECEIVED
 ↓
CLASSIFYING
 ↓
DISCOVERING_CONTEXT
 ↓
DESIGNING
 ↓
WAITING_DESIGN_APPROVAL
 ↓
IMPLEMENTING
 ↓
REVIEWING
 ↓
PREPARING_CHANGE
 ↓
WAITING_CODE_APPROVAL
 ↓
APPLYING_CHANGE
 ↓
BUILDING
 ↓
TESTING
 ↓
COMPLETED
```

May simplify or stub these states:

```text
SECURITY_REVIEWING
ANALYZING_DEPENDENCY
PARTIAL_APPROVAL
WAITING_RESOURCE
```

Workflow Engine owns transitions. LLM suggestions are not authoritative state changes.

Official state lives in PostgreSQL. Waiting workflows must survive server restart.

---

# 8. Context Entities

Persist at minimum:

```text
WORK_ITEM
WORKFLOW
WORKFLOW_TRANSITION
REQUIREMENT
DESIGN
APPROVAL
CHANGE_SET
BUILD_RESULT
TEST_RESULT
TOOL_CALL
```

Implement `context.get` and `context.put` first.

Do not implement advanced `context.query` unless needed for the minimum loop.

---

# 9. Agent Runtime

Shared runtime must provide:

```text
Agent Definition
Prompt Template
Context Builder
Model Client
Output Schema Validator
Result Writer
```

Call models only through Inference Gateway.

A single unified reasoning/coding model is acceptable in this phase.

Agents use model profiles, not hard-coded model IDs.

---

# 10. Tool Gateway

The server reaches the workspace only through Local Agent tools.

Gateway responsibilities:

- map domain calls to ToolRequest
- correlate call_id
- timeout / cancel
- structured ToolError
- no direct filesystem access

Phase 2 may still use a test client or harness if IDE is absent.

---

# 11. Approval

Sequence is always:

```text
PROPOSE → APPROVE → APPLY
```

Design approval and code-change approval are distinct.

An approval MUST be bound to the exact resource.

An approval MUST NOT be reused across change sets, file versions, workspaces, or projects.

Expired, consumed, rejected, or mismatched approvals MUST fail closed.

This phase may use a test/API approval adapter. The boundary must exist.

---

# 12. Required Tests

At minimum:

- workflow transitions are explicit and persisted
- WAITING_DESIGN_APPROVAL / WAITING_CODE_APPROVAL survive server restart
- design rejection stops implementation
- code rejection does not apply
- approval reuse is rejected
- expired approval is rejected
- agents cannot call apply
- context is stored by reference, not copied between agents
- Tool Gateway timeout/error is structured
- duplicate side-effect calls are protected

---

# 13. Definition of Done

Phase 2 is complete only when mock/simple project context can run:

```text
User Request
→ Design
→ Approval
→ Implementation
```

and all of the following are true:

1. Spring Boot server starts against PostgreSQL.
2. Work Item / Workflow state is persisted.
3. Workflow lease/recovery works after process restart in a waiting state.
4. Context Service supports get/put.
5. Tool Gateway can call Phase 1 Local Agent tools.
6. Senior Developer, Implementation, and Review exist as bounded agents.
7. Inference Gateway is the only model entry point.
8. Approval state is distinct for design and code.
9. Agents never apply workspace changes.
10. No IDE, Guide, Nexus, Redis, or HA is required for this phase.

---

# 14. Implementation Instructions for Coding Agents

Before changing code:

1. Inspect the repository.
2. Identify existing conventions from Phase 0 and Phase 1.
3. Produce an implementation plan.
4. List files/modules to create or modify.
5. Identify any conflict with architecture rules.
6. Do not redesign the Local Agent or protocol unless a conflict is reported.

During implementation:

- make small cohesive commits/changes
- add tests with each workflow capability
- prefer explicit state machines over implicit LLM control

After implementation:

1. Run unit and workflow tests.
2. Report architecture deviations.
3. Report remaining Phase 2 gaps.
4. Do not proceed to Phase 3 unless explicitly requested.
