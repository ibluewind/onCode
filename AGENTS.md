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
Before work  → read HANDOFF.md, issues, recent logs
             → write docs/plans/YYYY-MM-DD-작업계획.md
During work  → new issues in docs/issues/open.md
             → resolved issues moved to docs/issues/resolve.md
End of day   → docs/logs/YYYY-MM-DD.md
             → update docs/HANDOFF.md
```

Do not start implementation without a plan file.
Do not end a work day without a log and HANDOFF update.

---

## 2. Document Priority

When documents or existing code appear to conflict, use the following priority:

```text
1. docs/02_IMPLEMENTATION_RULES.md
2. protocol / API / DDL contracts
3. Current PHASE / TASK document
4. ADR
5. SPEC
6. Architecture overview
7. Existing implementation

```

Existing code is not automatically authoritative.

If existing code violates a higher-priority document, report it as an architectural deviation rather than silently following it.

---

## 3. Required References

Always read these before substantial implementation work:

```text
docs/01_ARCHITECTURE.md
docs/02_IMPLEMENTATION_RULES.md
docs/03_ADR_SUMMARY.md

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
- Public package registries are prohibited; internal Nexus is the dependency source.
- Hidden reasoning / chain-of-thought must not be persisted.
- High-risk operations must obey policy and approval requirements.

The full mandatory rules are defined in:

```text
docs/02_IMPLEMENTATION_RULES.md

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

At the end of a task, report:

- what was implemented
- files/modules changed
- tests executed and results
- architecture/spec deviations, if any
- unresolved issues
- remaining work in the current Phase

Do not automatically continue to the next Phase.