# Phase 07 Task — Dependency / Security

## 1. Goal

Connect closed-network dependency selection and offline security checks to design and apply.

This phase is Nexus/security focused.

Public package registries must not be used. All resolve/search goes through internal Nexus.

A vulnerable dependency request must not be auto-selected. Prefer a safe version, an alternative, a design change, or BLOCK.

Prerequisite: Phase 5 loop stable. Phase 6 Guide is preferred but must not block the critical path if already complete.

---

# 2. Required Reference Documents

Before implementation, read:

## 2.1 Overview

1. `../overview/01_ARCHITECTURE.md` — onCode Architecture Overview
2. `../overview/02_IMPLEMENTATION_RULES.md` — onCode Mandatory Implementation Rules
3. `../overview/03_ADR_SUMMARY.md` — onCode ADR Summary
4. `../overview/04_IMPLEMENTATION_ROADMAP.md` — onCode Implementation Roadmap
5. `PHASE_05.md` and `PHASE_06.md`
6. this task document

## 2.2 Relevant SPECs

Phase 7 Dependency / Security에 해당하는 `docs/spec` 문서:

1. `../spec/SPEC-11 onCode Nexus - Dependency - Vulnerability Integration Architecture v1.md`  
   — SPEC-11 onCode Nexus / Dependency / Vulnerability Integration Architecture v1  
   Nexus search/resolve, vulnerability, license. 이 phase의 핵심

2. `../spec/SPEC-08 onCode Security - Policy - Approval Model v1.md`  
   — SPEC-08 onCode Security / Policy / Approval Model v1  
   Phase 7에서 policy engine 강화, critical/high block

3. `../spec/SPEC-06 onCode Server Multi-Agent Architecture v1.md`  
   — SPEC-06 onCode Server Multi-Agent Architecture v1  
   Security Agent 분리 가능

4. `../spec/SPEC-03 onCode Context Storage Data Model - ERD v1.md`  
   — SPEC-03 onCode Context Storage Data Model / ERD v1  
   Dependency Decision persistence

5. `../spec/SPEC-13 onCode Audit - Logging - Observability Architecture v1.md`  
   — SPEC-13 onCode Audit / Logging / Observability Architecture v1  
   security/dependency audit

6. `../spec/SPEC-15 onCode MVP Scope - Implementation Roadmap v1.md`  
   — SPEC-15 onCode MVP Scope / Implementation Roadmap v1  
   Dependency/Security MVP, post-check, 완료 기준

Do not implement from later-phase SPECs in this task:

- SPEC-12 Authentication / RBAC — Phase 8 (권한은 최소)
- SPEC-14 HA / Production ops — Production
- SPEC-09 / SPEC-10 Guide — 이미 Phase 6, 이 task에서 재구현하지 않음

## 2.3 Relevant ADRs

- ADR-001 Server owns Nexus/security integration, not Local Agent
- ADR-005 PostgreSQL stores decisions and audit metadata
- ADR-011 Build-file changes still go through ProposedChanges + local actual diff + approval

---

# 3. In Scope

Implement:

```text
current dependency analysis
Nexus search
Nexus resolve
compatibility
vulnerability precheck
license check
dependency decision
approved build-file changes
actual resolution
dependency graph recheck
critical/high block
public registry block
Security Agent (may split from Review)
stronger policy engine
secret detection (basic)
audit search for security/dependency events
```

---

# 4. Out of Scope

Do not implement in this phase:

```text
public Maven Central / npmjs / PyPI as a runtime source
runtime server HA
full enterprise RBAC
Git push/rebase automation
vector DB
unbounded auto-upgrade of all dependencies
```

Do not expand scope unless explicitly instructed.

---

# 5. Recommended Module Structure

```text
server/

├─ dependency/
│  ├─ current/
│  ├─ nexus/
│  ├─ resolve/
│  └─ decision/
├─ security/
│  ├─ vulnerability/
│  ├─ license/
│  └─ secrets/
└─ agents/security/
```

Local Agent only applies approved build-file changes and may collect the local dependency tree. It does not choose versions.

---

# 6. New Dependency Workflow

```text
Design
→ Dependency Required
→ Current Dependency
→ Nexus
→ Security
→ Decision
→ Design Approval
```

Prefer existing project dependencies before adding new ones.

After actual resolve:

```text
Actual Dependency Graph
→ Security Recheck
```

Critical/High findings block. License policy failures block or require explicit human approval per policy.

Build-file edits follow:

```text
ProposedChanges
→ Local actual diff
→ Code approval
→ Atomic apply
```

---

# 7. Security Agent

This phase may split Security from Review.

Security Agent responsibilities:

- vulnerability precheck
- license check
- public registry block
- secret detection on proposed content
- post-resolve graph recheck

It still must not apply workspace changes.

---

# 8. Required Tests

At minimum:

- current dependencies are read from the sample Maven project
- Nexus search does not call public registries
- known-vulnerable coordinate is not auto-selected
- safe version is proposed when available
- BLOCK is returned when no safe option exists
- license deny-list is enforced
- post-resolve recheck can fail a previously accepted proposal
- build-file apply still requires approval and hash revalidation
- secret-like strings in proposed files are flagged
- dependency decisions are audited

---

# 9. Definition of Done

Phase 7 is complete only when a test case that requests a vulnerable dependency results in:

```text
자동 선택 금지
→ Safe Version
```

or:

```text
BLOCK
```

Also required:

1. Nexus is the only package search/resolve source.
2. Compatibility with project runtime/framework is considered.
3. Offline vulnerability DB is used.
4. License basic check works.
5. Actual resolve is rechecked.
6. Critical/High cannot be silently applied.
7. Security Agent or equivalent review gate exists.
8. No public registry dependency is introduced for runtime resolution.

---

# 10. Implementation Instructions for Coding Agents

Before changing code:

1. Confirm Phase 5 golden workflows still pass.
2. Produce an implementation plan for current-deps → Nexus → security → decision.
3. List files/modules to create or modify.
4. Identify any conflict with architecture rules.
5. Do not add public registry clients.

During implementation:

- make small cohesive commits/changes
- add tests with each security gate
- keep Local Agent execution-only

After implementation:

1. Run dependency/security tests including the vulnerable-coordinate case.
2. Report architecture deviations.
3. Report remaining Phase 7 gaps.
4. Do not proceed to Phase 8 unless explicitly requested.
