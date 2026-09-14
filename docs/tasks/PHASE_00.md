# Phase 00 Task — Protocol Foundation

## 1. Goal

Establish the shared onCode Tool Protocol and repository contract before any runtime component is built.

This phase is protocol-first.

Do not implement Local Agent tools, server workflow, or IDE UI.

The protocol must be transport-independent. gRPC, WebSocket, and JSON schemas are adapters around one logical contract.

Do this first. Changing IDs, error codes, or message shapes later is expensive.

---

# 2. Required Reference Documents

Before implementation, read:

## 2.1 Overview

1. `../overview/01_ARCHITECTURE.md` — onCode Architecture Overview
2. `../overview/02_IMPLEMENTATION_RULES.md` — onCode Mandatory Implementation Rules
3. `../overview/03_ADR_SUMMARY.md` — onCode ADR Summary
4. `../overview/04_IMPLEMENTATION_ROADMAP.md` — onCode Implementation Roadmap
5. this task document

## 2.2 Relevant SPECs

Phase 0 Protocol Foundation에 해당하는 `docs/spec` 문서:

1. `../spec/SPEC-01 onCode Tool Protocol v1.md`  
   — onCode Tool Protocol v1 Specification  
   공통 ID, error model, Tool Request/Response, versioning. 이 phase의 핵심

2. `../spec/SPEC-15 onCode MVP Scope - Implementation Roadmap v1.md`  
   — SPEC-15 onCode MVP Scope / Implementation Roadmap v1  
   Phase 0 산출물, ID 확정, repository 구조

Do not implement from later-phase SPECs in this task:

- SPEC-02 Workflow State Machine — Phase 2
- SPEC-03 Context Storage Data Model / ERD — Phase 2
- SPEC-04 Local Agent Architecture — Phase 1
- SPEC-05 Project Intelligence / PROJECT_INDEX Schema — Phase 4
- SPEC-06 Server Multi-Agent Architecture — Phase 2~5
- SPEC-07 IDE Extension Architecture — Phase 3
- SPEC-08 Security / Policy / Approval Model — Phase 1부터 부분 적용
- SPEC-09 Guide Ingestion / Preprocessing / Curation — Phase 6
- SPEC-10 Runtime Guide Repository / Retrieval — Phase 6
- SPEC-11 Nexus / Dependency / Vulnerability Integration — Phase 7
- SPEC-12 Authentication / Session / Authorization — Phase 8
- SPEC-13 Audit / Logging / Observability — P1 최소, Phase 8 강화
- SPEC-14 Deployment / HA / Operations — Production

## 2.3 Relevant ADRs

- ADR-003 Server ↔ Local Agent = gRPC bidirectional streaming — transport adapter only; protocol remains independent
- ADR-009 Monorepo initially — `protocol/` is a first-class package

---

# 3. In Scope

Implement:

```text
Common IDs
Error model
Protocol v1
protobuf schemas
JSON schemas where applicable
contract tests
protocol versioning
shared test fixtures
common DTO / domain identifiers
```

---

# 4. Out of Scope

Do not implement in this phase:

```text
Local Agent daemon
Tool handlers
Spring Boot server
Workflow engine
IDE extension
LLM / Inference Gateway
PostgreSQL persistence
Guide / Nexus
HA
```

Do not expand scope unless explicitly instructed.

---

# 5. Recommended Module Structure

```text
protocol/

├─ ids/
├─ errors/
├─ messages/
│  ├─ tool-request
│  ├─ tool-response
│  ├─ workflow-event
│  └─ approval
├─ schemas/
│  ├─ json/
│  └─ proto/
├─ versioning/
└─ tests/
   └─ fixtures/
```

Adapt if the repository already has a stronger established structure.

Do not reorganize unrelated code without need.

---

# 6. Required Identifiers

Fix these IDs now. Do not defer naming.

```text
message_id
call_id
session_id
project_id
workspace_id
work_item_id
workflow_id
approval_id
change_set_id
```

IDs must be:

- unique
- opaque to clients
- stable across transports
- documented with generation rules

---

# 7. Error Model

Define a structured, stable error model used by every later component.

At minimum the protocol must carry:

```text
error_code
message
retryable
details (non-sensitive)
```

Do not expose raw internal stack traces as protocol errors.

---

# 8. Protocol v1 Surface

Define logical messages for:

```text
ToolRequest
ToolResponse
ToolError
Heartbeat
Capabilities
Approval
WorkflowEvent
ProposedChangeSet
ActualDiff
```

JSON Schema is required for review and contract tests.

protobuf is required where gRPC will be used (Server ↔ Local Agent).

The JSON payload and protobuf mapping must represent the same logical protocol.

---

# 9. Versioning

Implement explicit protocol versioning.

```text
protocol_version
```

Rules:

- a client and server advertise supported versions
- unknown required fields fail closed
- additive optional fields may be ignored by older peers
- breaking changes require a new protocol version

---

# 10. Required Contract Tests

At minimum:

- every required ID is present and typed
- ToolRequest / ToolResponse round-trip JSON
- protobuf ↔ JSON mapping is lossless for required fields
- unknown protocol version is rejected
- structured error codes are stable
- fixtures cover success, validation failure, and transport-agnostic payloads

---

# 11. Definition of Done

Phase 0 is complete only when:

1. Shared protocol package exists in the repository.
2. Required IDs are named and documented.
3. Error codes are enumerated and tested.
4. JSON schemas exist for core messages.
5. protobuf schemas exist for Server ↔ Local Agent transport.
6. Contract tests pass.
7. Protocol version field is present and enforced.
8. No Local Agent tool handler or server workflow is implemented in this phase.
9. No public network dependency is introduced.

---

# 12. Implementation Instructions for Coding Agents

Before changing code:

1. Inspect the repository.
2. Identify existing conventions.
3. Produce an implementation plan.
4. List files/modules to create or modify.
5. Identify any conflict with architecture rules.
6. Do not redesign unrelated components.

During implementation:

- make small cohesive commits/changes
- add contract tests with each schema
- prefer explicit code over speculative abstraction

After implementation:

1. Run contract tests.
2. Report architecture deviations.
3. Report remaining Phase 0 gaps.
4. Do not proceed to Phase 1 unless explicitly requested.
