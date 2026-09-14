# Phase 05 Task — End-to-End Implementation Loop

## 1. Goal

Close the real coding loop on a Java/Maven sample project. This phase is the Technical MVP.

This phase is end-to-end focused. It wires Phase 1–4 capabilities into one repeatable workflow with bounded fix loops.

Do not add Guide, Nexus, HA, or extra agent roles unless required to finish the golden slices.

Prerequisite: Phase 0–4.

---

# 2. Required Reference Documents

Before implementation, read:

## 2.1 Overview

1. `../overview/01_ARCHITECTURE.md` — onCode Architecture Overview
2. `../overview/02_IMPLEMENTATION_RULES.md` — onCode Mandatory Implementation Rules
3. `../overview/03_ADR_SUMMARY.md` — onCode ADR Summary
4. `../overview/04_IMPLEMENTATION_ROADMAP.md` — onCode Implementation Roadmap
5. `PHASE_00.md` through `PHASE_04.md`
6. this task document

## 2.2 Relevant SPECs

Phase 5 End-to-End Implementation Loop에 해당하는 `docs/spec` 문서:

1. `../spec/SPEC-15 onCode MVP Scope - Implementation Roadmap v1.md`  
   — SPEC-15 onCode MVP Scope / Implementation Roadmap v1  
   E2E 흐름, fix loop, golden workflow, MVP 완료 기준. 이 phase의 핵심

2. `../spec/SPEC-02 onCode Workflow State Machine v1.md`  
   — SPEC-02 onCode Workflow State Machine v1  
   APPLYING_CHANGE / BUILDING / TESTING / fix loop 전이

3. `../spec/SPEC-04 onCode Local Agent Architecture v1.md`  
   — SPEC-04 onCode Local Agent Architecture v1  
   propose / actual diff / apply / build / test

4. `../spec/SPEC-05 onCode Project Intelligence - PROJECT_INDEX Schema v1.md`  
   — SPEC-05 onCode Project Intelligence / PROJECT_INDEX Schema v1  
   project.search → source read

5. `../spec/SPEC-06 onCode Server Multi-Agent Architecture v1.md`  
   — SPEC-06 onCode Server Multi-Agent Architecture v1  
   Senior Developer / Implementation / Review 최소 구성

6. `../spec/SPEC-07 onCode IDE Extension Architecture v1.md`  
   — SPEC-07 onCode IDE Extension Architecture v1  
   design/diff approval UX

7. `../spec/SPEC-08 onCode Security - Policy - Approval Model v1.md`  
   — SPEC-08 onCode Security / Policy / Approval Model v1  
   approval binding, stale apply 금지

8. `../spec/SPEC-13 onCode Audit - Logging - Observability Architecture v1.md`  
   — SPEC-13 onCode Audit / Logging / Observability Architecture v1  
   basic audit persist. P1 최소

Do not implement from later-phase SPECs in this task:

- SPEC-09 / SPEC-10 Guide — Phase 6
- SPEC-11 Nexus / Vulnerability — Phase 7
- SPEC-12 Authentication / RBAC — Phase 8
- SPEC-14 HA / Production ops — Production

## 2.3 Relevant ADRs

- ADR-010 Workflow DB is official state; waiting workflows resume after server restart
- ADR-011 ProposedChanges + Local actual diff + atomic apply
- ADR-012 project.search then workspace.read_files for current source
- ADR-007 Inference Gateway remains the only model path

---

# 3. In Scope

Implement the full loop:

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

Also implement:

```text
bounded build fix loop
bounded test fix loop
basic audit persistence
workflow persistence / resume
Java / Maven / JUnit first stack
VS Code first IDE
```

First stack:

```text
Java 17/21
Spring Boot
Maven
JUnit
```

---

# 4. Out of Scope

Do not implement in this phase:

```text
Guide Retrieval / Ingestion
Nexus / vulnerability / license
Security Agent
Git commit / push / merge / rebase
Node or Python as a required stack
vector DB
Kafka
Redis without measured need
Kubernetes as a development prerequisite
hunk-level approval
inline autocomplete
complete multi-language call graph
cross-repository implementation
large agent graph
```

Do not expand scope unless explicitly instructed.

---

# 5. Fix Loop

Build failure:

```text
Build Result
 ↓
Senior Developer
 ↓
Code Error 판단
 ↓
Relevant Source 추가 Read
 ↓
Implementation Fix
 ↓
Diff
 ↓
Approval
 ↓
Apply
 ↓
Build
```

Initial limits:

```text
Build Fix max 3
Test Fix max 3
Implementation Review max 5
```

Exceeding a limit returns control to the user. Do not loop forever.

---

# 6. MVP Review

Review Agent checks:

```text
Compile 가능성
Design Compliance
Existing Code Pattern
Obvious Error
Naming
Null/Error Handling
```

Detailed security review is Phase 7.

---

# 7. First Vertical Slices

These slices are mandatory.

## Slice 1

```text
Modify Greeter.java return value.
```

Validate: read, design, approval, propose, local diff, approval, apply, test.

## Slice 2

Add one method.

## Slice 3

Multi-file service change.

## Slice 4

Intentionally introduce/fix compile failure.

## Slice 5

Developer changes a file between proposal and approval; apply must fail stale.

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

# 9. Definition of Done

Phase 5 / Technical MVP is complete only when all are true:

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

Golden scenarios on the sample must succeed:

```text
기능 추가
기존 메서드 수정
Build Error 수정
Test Failure 수정
```

---

# 10. Implementation Instructions for Coding Agents

Before changing code:

1. Inspect Phases 0–4 outputs and identify gaps in the loop.
2. Produce an implementation plan focused on wiring, not new platforms.
3. List files/modules to create or modify.
4. Identify any conflict with architecture rules.
5. Do not start Guide, Nexus, or HA work.

During implementation:

- make small cohesive commits/changes
- add a golden-slice test with each closed gap
- keep agents minimal

After implementation:

1. Run unit, integration, and golden workflow tests.
2. Run the sample Maven project through the five slices.
3. Report architecture deviations.
4. Report remaining Technical MVP gaps.
5. Do not proceed to Phase 6 unless explicitly requested.
