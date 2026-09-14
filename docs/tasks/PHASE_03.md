# Phase 03 Task — IDE Interaction

## 1. Goal

Connect a real developer UI to the Phase 2 server workflow and Phase 1 Local Agent.

This phase is IDE-interaction focused.

First IDE is VS Code unless organizational constraints require otherwise.

The extension is a thin client. It must not read, write, build, test, or index the workspace itself.

Prerequisite: Phase 0 protocol, Phase 1 Local Agent, Phase 2 server workflow.

---

# 2. Required Reference Documents

Before implementation, read:

## 2.1 Overview

1. `../overview/01_ARCHITECTURE.md` — onCode Architecture Overview
2. `../overview/02_IMPLEMENTATION_RULES.md` — onCode Mandatory Implementation Rules
3. `../overview/03_ADR_SUMMARY.md` — onCode ADR Summary
4. `../overview/04_IMPLEMENTATION_ROADMAP.md` — onCode Implementation Roadmap
5. `PHASE_01.md` and `PHASE_02.md`
6. this task document

## 2.2 Relevant SPECs

Phase 3 IDE Interaction에 해당하는 `docs/spec` 문서:

1. `../spec/SPEC-07 onCode IDE Extension Architecture v1.md`  
   — SPEC-07 onCode IDE Extension Architecture v1  
   Thin client, chat/progress/approval/diff. 이 phase의 핵심

2. `../spec/SPEC-01 onCode Tool Protocol v1.md`  
   — onCode Tool Protocol v1 Specification  
   IDE ↔ Local Agent / approval events

3. `../spec/SPEC-02 onCode Workflow State Machine v1.md`  
   — SPEC-02 onCode Workflow State Machine v1  
   WAITING_DESIGN_APPROVAL / WAITING_CODE_APPROVAL UI 매핑

4. `../spec/SPEC-04 onCode Local Agent Architecture v1.md`  
   — SPEC-04 onCode Local Agent Architecture v1  
   IDE는 Local Agent에만 연결. 파일 작업 위임

5. `../spec/SPEC-08 onCode Security - Policy - Approval Model v1.md`  
   — SPEC-08 onCode Security / Policy / Approval Model v1  
   Design/code approve·reject, approval binding

6. `../spec/SPEC-15 onCode MVP Scope - Implementation Roadmap v1.md`  
   — SPEC-15 onCode MVP Scope / Implementation Roadmap v1  
   Phase 3 기능, 제외 항목, 완료 기준

Do not implement from later-phase SPECs in this task:

- SPEC-05 Project Intelligence — Phase 4
- SPEC-09 / SPEC-10 Guide — Phase 6
- SPEC-11 Nexus / Security — Phase 7
- SPEC-12 Authentication / Session / Authorization — Phase 8 (local session token만)
- SPEC-14 Deployment / HA — Production

## 2.3 Relevant ADRs

- ADR-004 IDE ↔ Local Agent = localhost WebSocket + JSON
- ADR-003 Server ↔ Local Agent = gRPC; IDE does not talk to the server as the execution path
- ADR-011 Local Agent-generated actual diff is what the user approves

---

# 3. In Scope

Implement:

```text
VS Code extension
chat
current file context
selection context
progress
question UI
design review
design approve/reject
diff review
code approve/reject
build/test result
error notification
reconnect
```

Initial IDE ↔ Local Agent transport:

```text
localhost WebSocket + JSON
```

---

# 4. Out of Scope

Do not implement in this phase:

```text
Inline Completion
Code Lens
Hunk Approval
Advanced History
Multi-root UI
Architecture Visualization
Eclipse / IntelliJ (unless org standard requires it first)
Guide UI
Nexus UI
enterprise IdP login
```

Do not expand scope unless explicitly instructed.

---

# 5. Recommended Module Structure

```text
ide/vscode/

├─ extension/
├─ chat/
├─ progress/
├─ approval/
├─ diff/
├─ ipc/
└─ tests/
```

Keep the extension thin. Execution stays in Local Agent. Orchestration stays on the server.

---

# 6. IPC Security

Local WebSocket must:

- bind to loopback only
- require a local session token
- validate origin/client
- rotate the token on Local Agent restart

The IDE must not store long-lived central credentials in this phase.

---

# 7. Approval UX

Design review and code/diff review are separate screens/states.

The diff shown to the user MUST be the Local Agent actual diff, not a model-generated diff.

Approve and reject must bind to:

```text
approval_id
resource identity
diff hash / design identity
workspace_id
work_item_id
```

Reject must stop the workflow. Approve must not apply by itself; apply remains a Local Agent tool after server validation.

---

# 8. Reconnect

The extension must reconnect to Local Agent after:

- IDE reload
- Local Agent restart
- transient IPC failure

On reconnect, restore in-progress work item progress and pending approval, if any.

---

# 9. Required Tests

At minimum:

- chat request creates/resumes a work item
- progress updates render for workflow states
- design approve and reject
- code/diff approve and reject
- rejected design does not show apply
- displayed diff matches Local Agent actual diff
- IPC refuses non-loopback binds in tests
- missing/invalid local token is rejected
- reconnect restores pending approval
- extension does not write workspace files directly

---

# 10. Definition of Done

Phase 3 is complete only when a real IDE can complete:

```text
사용자 요청
→ Server Design
→ IDE 승인
```

and all of the following are true:

1. VS Code extension installs and connects to Local Agent on localhost.
2. Chat, progress, and question UI work.
3. Design review approve/reject works.
4. Diff review shows Local Agent actual diff.
5. Code approve/reject works.
6. Build/test results can be displayed when present.
7. Reconnect works.
8. Extension does not perform workspace file I/O, build, or test.
9. No inline completion, hunk approval, or extra IDEs are required.

---

# 11. Implementation Instructions for Coding Agents

Before changing code:

1. Inspect the repository.
2. Identify existing Local Agent IPC and server approval APIs.
3. Produce an implementation plan.
4. List files/modules to create or modify.
5. Identify any conflict with architecture rules.
6. Do not move execution logic into the extension.

During implementation:

- make small cohesive commits/changes
- add tests with each UI capability
- keep the client thin

After implementation:

1. Run extension/IPC tests.
2. Exercise design approval in the IDE.
3. Report architecture deviations.
4. Report remaining Phase 3 gaps.
5. Do not proceed to Phase 4 unless explicitly requested.
