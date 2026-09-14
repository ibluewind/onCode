# Phase 04 Task — Project Intelligence

## 1. Goal

Give the server an efficient, local-first understanding of the project so `project.search` can return relevant files without uploading the whole workspace.

This phase is Project Intelligence focused.

Do not wait for a complete semantic call graph. Relevant file discovery comes first.

The Local Agent owns indexing. The central catalog stores delta-synced search metadata. The server still reads current source only through Local Agent file tools.

Prerequisite: Phase 0–3. Phase 2 may have used mock project context; replace that with real search in this phase.

---

# 2. Required Reference Documents

Before implementation, read:

## 2.1 Overview

1. `../overview/01_ARCHITECTURE.md` — onCode Architecture Overview
2. `../overview/02_IMPLEMENTATION_RULES.md` — onCode Mandatory Implementation Rules
3. `../overview/03_ADR_SUMMARY.md` — onCode ADR Summary
4. `../overview/04_IMPLEMENTATION_ROADMAP.md` — onCode Implementation Roadmap
5. `PHASE_01.md` — Local Agent ownership of workspace tools
6. this task document

## 2.2 Relevant SPECs

Phase 4 Project Intelligence에 해당하는 `docs/spec` 문서:

1. `../spec/SPEC-05 onCode Project Intelligence - PROJECT_INDEX Schema v1.md`  
   — SPEC-05 onCode Project Intelligence / PROJECT_INDEX Schema v1  
   PROJECT_INDEX, scanner, symbol, delta sync. 이 phase의 핵심

2. `../spec/SPEC-04 onCode Local Agent Architecture v1.md`  
   — SPEC-04 onCode Local Agent Architecture v1  
   Local Agent가 indexing / search 실행

3. `../spec/SPEC-01 onCode Tool Protocol v1.md`  
   — onCode Tool Protocol v1 Specification  
   project.search 및 source read tool

4. `../spec/SPEC-15 onCode MVP Scope - Implementation Roadmap v1.md`  
   — SPEC-15 onCode MVP Scope / Implementation Roadmap v1  
   Phase 4 MVP, Java 우선, 완료 기준

Do not implement from later-phase SPECs in this task:

- SPEC-09 / SPEC-10 Guide — Phase 6
- SPEC-11 Nexus / Vulnerability — Phase 7 (dependency metadata 최소만)
- SPEC-12 Authentication — Phase 8
- SPEC-14 Deployment / HA — Production

## 2.3 Relevant ADRs

- ADR-006 Hybrid parser: Tree-sitter + language-specific semantic adapters
- ADR-012 Hybrid local-index / central-catalog; no vector DB for MVP
- ADR-002 Local Agent = Go; parsers may be adapter-based and not all Go-native

---

# 3. In Scope

Implement:

```text
Project Detector
Workspace Scanner
File Hash
Language Detection
Basic Parser
File Summary
Symbol Extraction
import / relation extraction (basic)
dependency metadata
PROJECT_INDEX
central catalog delta sync
file / symbol search
incremental watcher
```

First language:

```text
Java
```

Framework interpretation (Spring / eGovFramework) belongs in analyzers, not the parser core.

---

# 4. Out of Scope

Do not implement in this phase:

```text
vector DB
complete call graph for all languages
LLM file summaries as a prerequisite
Guide retrieval
Nexus resolve
cross-repository indexing
large-scale refactoring
inline autocomplete
```

Do not expand scope unless explicitly instructed.

---

# 5. Recommended Module Structure

```text
local-agent/internal/intelligence/

├─ detector/
├─ scanner/
├─ parser/
│  ├─ treesitter/
│  └─ java/
├─ index/
├─ watcher/
└─ sync/

server/project-catalog/
```

---

# 6. Priority

Implement in this order:

```text
relevant file discovery
>
symbol accuracy
>
summary quality
>
complete call graph
```

Heuristic summaries are enough initially:

```text
File Name
Package
Class
Methods
Javadoc
```

Add LLM summaries only after structural index is stable.

---

# 7. Minimum Index Schema

```text
Project
Module
File
Symbol
Dependency
Revision
Hash
```

Relations start at basic import level.

Do not rescan the whole project on every request. File watcher and incremental index are required from the start.

---

# 8. Search Path

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

```text
exact path/symbol
keyword
PostgreSQL FTS
trigram
relation expansion
```

The server MUST still read actual source through Local Agent. Catalog metadata is not a substitute for current file content.

---

# 9. Required Tests

At minimum:

- Java sample project is detected
- file hashes are stable
- symbols are extracted for classes/methods
- imports are recorded at basic level
- incremental watcher updates index after a file change
- full rescan is not required for a single-file edit
- delta sync updates central catalog
- `project.search` for "로그인 처리" class of queries returns related files on the sample
- search does not upload full source as the index payload
- excluded dirs (`.git`, `target`, `node_modules`) are skipped

---

# 10. Definition of Done

Phase 4 is complete only when the server can search a sample Java project for:

```text
로그인 처리
```

and retrieve candidates such as:

```text
LoginController
AuthService
UserRepository
```

and all of the following are true:

1. Workspace scanner and Tree-sitter/Java parser run on Local Agent.
2. PROJECT_INDEX is generated and incrementally updated.
3. Central catalog receives deltas.
4. `project.search` returns relevant files.
5. Actual source is still read via `workspace.read_files`.
6. No vector DB is introduced.
7. Call-graph completeness is not a gate.
8. Node/Python parsers are not required unless already cheap to stub.

---

# 11. Implementation Instructions for Coding Agents

Before changing code:

1. Inspect Local Agent and server catalog code.
2. Produce an implementation plan.
3. List files/modules to create or modify.
4. Identify any conflict with architecture rules.
5. Do not move indexing onto the central server.

During implementation:

- make small cohesive commits/changes
- add tests with each index capability
- keep Java quality ahead of extra languages

After implementation:

1. Run index unit tests and sample search.
2. Report architecture deviations.
3. Report remaining Phase 4 gaps.
4. Do not proceed to Phase 5 unless explicitly requested.
