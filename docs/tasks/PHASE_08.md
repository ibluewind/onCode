# Phase 08 Task — Authentication / Operations / Hardening

## 1. Goal

Harden onCode for pilot and production: identity, authorization, observability, and operational readiness.

This phase is auth/ops focused.

MVP feature completeness is Phase 5. This phase does not reopen Guide or Nexus scope. It makes the existing system operable, auditable, and recoverable.

Prerequisite: Phase 5 Technical MVP. Phases 6–7 should be complete if those products are in the target pilot.

---

# 2. Required Reference Documents

Before implementation, read:

## 2.1 Overview

1. `../overview/01_ARCHITECTURE.md` — onCode Architecture Overview
2. `../overview/02_IMPLEMENTATION_RULES.md` — onCode Mandatory Implementation Rules
3. `../overview/03_ADR_SUMMARY.md` — onCode ADR Summary
4. `../overview/04_IMPLEMENTATION_ROADMAP.md` — onCode Implementation Roadmap
5. `PHASE_05.md`
6. this task document

## 2.2 Relevant SPECs

Phase 8 Authentication / Operations / Hardening에 해당하는 `docs/spec` 문서:

1. `../spec/SPEC-12 onCode Authentication - Session - Authorization Architecture v1.md`  
   — SPEC-12 onCode Authentication / Session / Authorization Architecture v1  
   IdP, session, Local Agent registration, RBAC. 이 phase의 핵심

2. `../spec/SPEC-13 onCode Audit - Logging - Observability Architecture v1.md`  
   — SPEC-13 onCode Audit / Logging / Observability Architecture v1  
   audit, metrics, health, dashboards. Phase 8에서 강화

3. `../spec/SPEC-14 onCode Deployment - HA - Operations Architecture v1.md`  
   — SPEC-14 onCode Deployment / HA / Operations Architecture v1  
   runtime HA 준비, backup/restore, rolling deploy

4. `../spec/SPEC-08 onCode Security - Policy - Approval Model v1.md`  
   — SPEC-08 onCode Security / Policy / Approval Model v1  
   project membership과 tool/approval scope

5. `../spec/SPEC-15 onCode MVP Scope - Implementation Roadmap v1.md`  
   — SPEC-15 onCode MVP Scope / Implementation Roadmap v1  
   Phase 8 auth/observability/deployment, MVP vs Production 구분

Do not re-implement earlier SPECs in this task except to wire identity, audit, and ops around them.

## 2.3 Relevant ADRs

- ADR-001 Spring Security on the modular monolith
- ADR-005 PostgreSQL remains workflow/session SSOT; Redis only if measured need
- ADR-008 shared artifact storage behind ArtifactStorage
- ADR-010 workflow lease for multi-instance runtime
- ADR-003 Local Agent initiates outbound persistent connection

---

# 3. In Scope

Implement:

```text
internal IdP integration
session
Local Agent registration
project membership / RBAC
workspace scope
audit
structured logging
metrics
health checks
runtime HA preparation
backup / restore
operational dashboards
agent version enforcement
operational runbooks
recovery drills (as tests/procedures)
```

Production-oriented additions from the roadmap:

```text
2+ runtime servers (prepare, even if MVP still runs one)
PostgreSQL backup / HA
shared artifact storage
central monitoring / alerts
rolling deployment
```

---

# 4. Out of Scope

Do not implement in this phase:

```text
new coding-loop features
vector DB
Kafka as a default
Kubernetes as a development prerequisite
hunk-level approval
inline autocomplete
cross-repository implementation
rewriting Local Agent tools
```

Redis is allowed only for connection registry, pub/sub, locks, or session acceleration after measured need. Redis MUST NOT become workflow SSOT.

Do not expand scope unless explicitly instructed.

---

# 5. Recommended Module Structure

```text
server/

├─ auth/
├─ session/
├─ rbac/
├─ agent-registry/
├─ audit/
├─ observability/
└─ ops/

deployment/
├─ runbooks/
└─ health/
```

---

# 6. Authentication and Authorization

Users authenticate through an internal IdP.

IDE must not store long-lived central credentials.

Local Agent registration binds:

```text
user
agent
workspace
project
session
```

RBAC must separate:

```text
project membership
workspace scope
work item access
approval rights
admin / operator rights
```

Another user's context MUST NOT leak into a workflow.

IdP outage must fail closed for new sessions and have a documented degraded mode for in-flight work.

---

# 7. Observability

Strengthen SPEC-13 minimums into operable telemetry:

```text
Audit
Structured Log
Metrics
Workflow Timeline
Local Agent Health
```

Do not log:

```text
full file content
credentials
tokens
private keys
chain-of-thought
```

Audit write failure must fail closed for security-sensitive actions.

---

# 8. Operations / HA Preparation

Prepare, do not overbuild:

- runtime servers can restart and resume WAITING_APPROVAL workflows
- Local Agent persistent connection registry can move toward multi-node routing
- PostgreSQL backup/restore is documented and tested
- artifact storage is shared, not node-local
- health: liveness vs readiness
- agent version policy can block incompatible Local Agents
- rolling deploy / drain procedure exists
- recovery drills exist as runbooks plus automated tests where feasible

Full multi-site and Kubernetes are not required to finish this phase if a documented production profile exists.

---

# 9. Required Tests

At minimum:

- IdP login creates a server session
- expired session is rejected
- Local Agent registration binds user/workspace/project
- other-project work item access is denied
- RBAC denies unauthorized approval
- waiting workflow resumes after server restart
- Local Agent disconnect/reconnect restores routing
- audit records auth, approval, apply, admin actions
- secrets/tokens are not present in application logs
- backup restore recovers workflow/approval data
- incompatible agent version is rejected
- health endpoints distinguish liveness and readiness

---

# 10. Definition of Done

Phase 8 is complete only when:

1. Internal IdP authentication works.
2. Sessions are server-authoritative.
3. Local Agent registration and reconnect are scoped.
4. Project membership/RBAC is enforced.
5. Audit, structured logs, metrics, and health checks exist.
6. Workflow state survives restart and is ready for multi-instance lease.
7. Backup/restore of official PostgreSQL state is proven.
8. Shared artifact storage is used.
9. Agent version enforcement exists.
10. Operational dashboards/alerts cover Local Agent, workflow, and inference health.
11. Runbooks exist for restart, drain, backup restore, and IdP outage.
12. Redis/Kafka/Kubernetes are not introduced without measured need.

Distinguish:

```text
MVP        = 기능 동작 검증 (Phase 5)
Pilot      = 실제 개발자 일부 사용 (auth + ops minimum)
Production = HA / Audit / Security / 운영 체계
```

This phase must reach Pilot-ready and prepare Production. It does not require every SPEC-14 production profile item if remaining items are listed as explicit gaps.

---

# 11. Implementation Instructions for Coding Agents

Before changing code:

1. Inspect existing session stubs from Phase 2 and audit stubs from Phase 1/5.
2. Produce an implementation plan for identity first, then observability, then HA preparation.
3. List files/modules to create or modify.
4. Identify any conflict with architecture rules.
5. Do not make Redis the workflow source of truth.

During implementation:

- make small cohesive commits/changes
- add tests with each auth/ops capability
- keep coding-loop behavior unchanged except for identity/audit wrapping

After implementation:

1. Run auth, RBAC, resume, and backup-restore tests.
2. Report architecture deviations.
3. Report remaining Pilot vs Production gaps.
4. Do not start unrelated feature work unless explicitly requested.
