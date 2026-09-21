# onCode Agent Instructions

## 1. Required Work Procedure

For every implementation task, follow this sequence:

```text
Developer Request
   ↓
Identify current PHASE / TASK
   ↓
Read relevant IMPLEMENTATION_RULES
   ↓
Read relevant ADRs
   ↓
Read relevant Protocol / API / DDL contracts
   ↓
Read only the SPECs required for detailed clarification
   ↓
Inspect the current repository and implementation
   ↓
Produce an implementation plan
   ↓
Implement only the approved/current task scope
   ↓
Review implementation against relevant SPEC / ADR / rules
   ↓
Run required tests
   ↓
Report results, deviations, and remaining gaps

```

Do not skip the architecture review step.

Work records are mandatory. Follow `.cursor/rules/work-process.mdc`:

```text
Before work  → read HANDOFF.md, issues, recent logs, recent docs/results/
             → write docs/plans/YYYY-MM-DD-작업계획.md
During work  → new issues in docs/issues/open.md
             → resolved issues moved to docs/issues/resolve.md
After a slice (tests run) → docs/results/YYYY-MM-DD-구현결과.md
End of day   → docs/logs/YYYY-MM-DD.md
             → update docs/HANDOFF.md
```

Do not start implementation without a plan file.
Do not treat a slice as done until `docs/results/` records what was implemented, how it was tested, and the result.
Do not end a work day without a log and HANDOFF update.

---

## 2. Document Priority

When documents or existing code appear to conflict, use the following priority:

```text
1. docs/overview/02_IMPLEMENTATION_RULES.md (runtime product behavior, not this repo's toolchain)
2. protocol / API / DDL contracts
3. Current PHASE / TASK document
4. ADR
5. SPEC
6. Architecture overview
7. Existing implementation

```

Existing code is not automatically authoritative.

`02_IMPLEMENTATION_RULES.md` describes what onCode must enforce on **user workspaces** once the platform runs. It is not the build/dependency policy of this repository.

If existing code violates a higher-priority document, report it as an architectural deviation rather than silently following it.

---

## 3. Required References

Always read these before substantial implementation work:

```text
docs/overview/01_ARCHITECTURE.md
docs/overview/02_IMPLEMENTATION_RULES.md
docs/overview/03_ADR_SUMMARY.md

```

Then read the current task document under:

```text
docs/tasks/

```

Read relevant detailed SPECs only as needed from:

```text
docs/spec/

```

Read protocol contracts from:

```text
protocol/

```

---

## 4. Scope Discipline

Implement only the requested/current Phase or Task.

Do not implement future phases unless explicitly requested.

Do not redesign unrelated components.

Do not introduce infrastructure or dependencies solely because they may be useful later.

Examples of technologies that require a demonstrated need before introduction:

```text
Redis
Kafka
Vector DB
Kubernetes
additional microservices
arbitrary shell execution

```

---

## 5. Architecture Invariants

The following rules are mandatory:

- The Central Server MUST NOT directly modify developer workspace files.
- Workspace operations MUST go through the Local Agent.
- Implementation Agents produce ProposedChanges only.
- The Local Agent generates the actual diff against current workspace files.
- Code changes follow `PROPOSE → APPROVE → APPLY`.
- Stale workspace content MUST NOT be force-applied.
- Shared agent context is stored in Context Storage and referenced via context refs.
- PostgreSQL-backed workflow state is the source of truth.
- Build, test, Git, and local execution occur through the Local Agent.
- Hidden reasoning / chain-of-thought must not be persisted.
- High-risk operations must obey policy and approval requirements.

The platform must **implement** the user-workspace rules in `docs/overview/02_IMPLEMENTATION_RULES.md` (including Nexus-only dependency resolve for projects onCode changes). Those rules do not govern how this repository is built.

The full mandatory runtime rules are defined in:

```text
docs/overview/02_IMPLEMENTATION_RULES.md

```

---

## 6. Before Writing Code

Before modifying code for a non-trivial task:

1. Inspect the relevant repository/module.
2. Identify existing implementation and conventions.
3. Identify the current task scope.
4. Read the relevant contracts and ADRs.
5. Identify the SPEC sections needed for clarification.
6. Produce a concise implementation plan.
7. List the files/modules expected to change.
8. Identify any conflict with the documented architecture.

Do not silently resolve architectural conflicts by inventing a new design.

---

## 7. During Implementation

- Keep changes cohesive and limited to the current task.
- Follow existing code conventions unless they violate higher-priority architecture documents.
- Add or update tests with the implementation.
- Prefer explicit behavior over speculative abstraction.
- Preserve protocol compatibility.
- Do not reuse removed Protocol Buffer field numbers.
- Do not bypass approval or policy boundaries for convenience.

Coding conventions for **this repository** (not user-workspace IMPLEMENTATION_RULES):

```text
.cursor/rules/comments.mdc           — class/type and function comments
.cursor/rules/sql-naming.mdc         — PostgreSQL table/column names + COMMENT ON
.cursor/rules/java-naming.mdc        — server (Google Java Style)
.cursor/rules/go-naming.mdc          — local-agent, protocol (Effective Go)
.cursor/rules/protocol-naming.mdc    — JSON/proto snake_case fields
.cursor/rules/typescript-naming.mdc  — future IDE
```

Rules are **per language**, not per component. Map wire names (`work_item_id`) to the language idiom (`workItemId` / `WorkItemID`).

---

## 8. Architecture Review

After implementation, review the changes against:

```text
IMPLEMENTATION_RULES
Relevant ADRs
Relevant SPECs
Protocol/API contracts
Current TASK acceptance criteria

```

Explicitly check for:

1. requirement violations
2. missing requirements
3. architectural drift
4. security boundary violations
5. scope creep
6. protocol incompatibility
7. missing failure handling

Fix violations before considering the task complete.

---

## 9. Testing

Run the tests required by the current task.

At minimum, where applicable:

```text
Unit Tests
Contract Tests
Integration Tests
Relevant Build
Relevant Test Suite

```

Security-sensitive or side-effecting functionality must include negative/failure-path tests.

Examples:

```text
stale workspace
invalid approval
path escape
hash mismatch
duplicate execution
timeout
rollback failure

```

---

## 10. Completion Report

Write the same information to `docs/results/YYYY-MM-DD-구현결과.md` (same date: append). Then summarize in chat.

Must include:

- what was implemented, files/modules changed, what was explicitly out of scope
- how tests were run (command, unit vs integration, JDK, Maven settings, DB — Testcontainers vs local compose)
- each test case and PASS/FAIL
- command output totals (Tests run / Failures / Errors / Skipped)
- intermediate failures and the fix
- architecture/spec deviations, if any
- unresolved issues
- remaining work in the current Phase

Do not automatically continue to the next Phase.