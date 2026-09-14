# onCode ADR Summary

## ADR-001 Central Server Technology

**Decision:** Java + Spring Boot.

Rationale:

- enterprise-grade server ecosystem
- workflow / transaction support
- Spring Security
- WebSocket / API integration
- PostgreSQL integration
- operational maturity

Initial form: **Modular Monolith**.

---

## ADR-002 Local Agent Technology

**Decision:** Go.

Rationale:

- cross-platform single binary
- good fit for long-running daemon
- filesystem/process/network control
- low deployment friction
- gRPC support
- good Windows/macOS/Linux story

Parser implementation is adapter-based and does not require all parsers to be Go-native.

---

## ADR-003 Server ↔ Local Agent Communication

**Decision:** gRPC bidirectional streaming.

Logical protocol remains transport-independent.

```text
onCode Tool Protocol
→ gRPC transport adapter
```

The Local Agent initiates an outbound persistent connection.

---

## ADR-004 IDE ↔ Local Agent IPC

**Decision:** localhost WebSocket + JSON.

Security requirements:

- loopback binding only
- local session token
- origin/client validation
- token rotation on agent restart

---

## ADR-005 Persistence / Session / Distributed State

**Decision:** PostgreSQL-only for MVP/Pilot.

PostgreSQL is the source of truth for:

- workflow
- context metadata
- approvals
- sessions
- project catalog
- guide runtime
- audit metadata

Redis may be introduced later for:

- fast distributed connection registry
- pub/sub
- locks
- session acceleration

Redis must not become the workflow SSOT.

---

## ADR-006 Project Intelligence Parser Architecture

**Decision:** Hybrid parser architecture.

```text
Tree-sitter
→ common structural parsing

Language-specific semantic adapters
→ deeper resolution only where needed
```

Examples:

- Java semantic adapter
- TypeScript compiler/language-service adapter

Spring/eGovFramework interpretation belongs in framework analyzers, not the parser itself.

---

## ADR-007 LLM Inference Architecture

**Decision:**

```text
Agent Runtime
→ Inference Gateway
→ vLLM
```

Agents use model profiles, not hard-coded model IDs.

Example profiles:

```text
reasoning-high
coding
review
utility
```

Inference Gateway owns:

- routing
- timeout
- queue
- concurrency
- health
- metrics
- fallback policy

---

## ADR-008 Artifact Storage Architecture

**Decision:** Shared NFS/NAS initially behind an `ArtifactStorage` abstraction.

Logical URIs:

```text
artifact://build/EXEC-100/log
artifact://guide/DOC-100/original
```

Future backend may be S3-compatible internal object storage.

---

## ADR-009 Repository Strategy

**Decision:** Monorepo initially.

Recommended structure:

```text
oncode/
├─ server/
├─ local-agent/
├─ ide/
├─ protocol/
├─ guide/
├─ deployment/
├─ samples/
├─ docs/
└─ tests/
```

Split into multiple repositories only after protocol and component boundaries stabilize.

---

## ADR-010 Workflow Execution Model

**Decision:** PostgreSQL persistent state machine + workflow lease.

```text
Workflow DB
→ runnable workflow
→ acquire lease
→ execute handler
→ persist transition
→ release lease
```

Future Redis/broker can provide wake-up signaling but does not own official state.

---

## ADR-011 Change Application Model

**Decision:** Structured ProposedChanges + Local Agent-generated actual diff + atomic apply.

Do not trust an LLM-generated unified diff as the final approval/apply artifact.

```text
Implementation Agent
→ ProposedChanges
→ Local Agent
→ Actual Diff
→ User Approval
→ Hash Revalidation
→ Atomic Apply
```

---

## ADR-012 Project Search Architecture

**Decision:** Hybrid local-index / central-catalog architecture.

```text
Local Project Intelligence
→ PROJECT_INDEX
→ Delta Sync
→ Central Project Catalog
→ project.search
→ candidate files
→ workspace.read_files
→ current source
```

Initial central search:

- exact path/symbol
- keyword
- PostgreSQL FTS
- trigram
- relation expansion

No vector DB required for MVP.
