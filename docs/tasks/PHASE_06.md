# Phase 06 Task — Guide

## 1. Goal

Attach organization/project development guides to the stable coding loop.

This phase is guide-focused and split into 6A Ingestion and 6B Runtime Retrieval.

Do not start Guide work until the Phase 5 Technical MVP loop is stable. Guide search sophistication must not delay the critical path.

No vector DB. Runtime search is PostgreSQL-based.

Prerequisite: Phase 5 Technical MVP.

---

# 2. Required Reference Documents

Before implementation, read:

## 2.1 Overview

1. `../overview/01_ARCHITECTURE.md` — onCode Architecture Overview
2. `../overview/02_IMPLEMENTATION_RULES.md` — onCode Mandatory Implementation Rules
3. `../overview/03_ADR_SUMMARY.md` — onCode ADR Summary
4. `../overview/04_IMPLEMENTATION_ROADMAP.md` — onCode Implementation Roadmap
5. `PHASE_05.md` — the loop Guide must attach to
6. this task document

## 2.2 Relevant SPECs

Phase 6 Guide에 해당하는 `docs/spec` 문서:

1. `../spec/SPEC-09 onCode Guide Ingestion - Preprocessing - Curation Architecture v1.md`  
   — SPEC-09 onCode Guide Ingestion / Preprocessing / Curation Architecture v1  
   6A upload, conversion, curation, publish

2. `../spec/SPEC-10 onCode Runtime Guide Repository - Retrieval Architecture v1.md`  
   — SPEC-10 onCode Runtime Guide Repository / Retrieval Architecture v1  
   6B release pin, FTS, mandatory rules

3. `../spec/SPEC-03 onCode Context Storage Data Model - ERD v1.md`  
   — SPEC-03 onCode Context Storage Data Model / ERD v1  
   Guide reference를 workflow context에 저장

4. `../spec/SPEC-06 onCode Server Multi-Agent Architecture v1.md`  
   — SPEC-06 onCode Server Multi-Agent Architecture v1  
   Design/Implementation/Review가 동일 Guide를 참조

5. `../spec/SPEC-15 onCode MVP Scope - Implementation Roadmap v1.md`  
   — SPEC-15 onCode MVP Scope / Implementation Roadmap v1  
   Phase 6 pipeline, console, 완료 기준

Do not implement from later-phase SPECs in this task:

- SPEC-11 Nexus / Vulnerability — Phase 7
- SPEC-12 Authentication — Phase 8 (admin console은 최소)
- SPEC-14 HA / ingestion scale-out — Production 수준 배포는 후순위

## 2.3 Relevant ADRs

- ADR-005 PostgreSQL is SSOT for guide runtime; no Redis/vector requirement
- ADR-008 ArtifactStorage for original guide files (`artifact://guide/...`)
- ADR-007 Agents retrieve guides through server tools, not ad-hoc prompt stuffing

---

# 3. In Scope

## 6A Ingestion

```text
PDF
DOCX
PPTX
HWP/HWPX
Markdown
Docling
LibreOffice fallback
normalized markdown
structured sections
summary / keyword extraction
curation console
publish
```

Pipeline:

```text
Upload
→ Conversion
→ Markdown
→ Section Split
→ Summary
→ Keyword
→ Admin Review
→ Publish
```

Console minimum:

```text
Upload
Status
Markdown Preview
Section Edit
Category
Summary
Keyword
Publish
```

## 6B Runtime Retrieval

```text
release pinning
metadata filter
keyword
FTS
trigram
technology / category / version
mandatory rules
guide references
```

Mandatory rule types that must exist:

```text
PROJECT_RULE
ORGANIZATION_RULE
REQUIRED
SECURITY_REQUIRED
```

---

# 4. Out of Scope

Do not implement in this phase:

```text
vector DB
Nexus / vulnerability
Security Agent
runtime HA
advanced multi-site guide replication
unbounded guide context injection into prompts
```

Do not expand scope unless explicitly instructed.

---

# 5. Recommended Module Structure

```text
guide/

├─ ingestion/
│  ├─ convert/
│  ├─ normalize/
│  ├─ section/
│  └─ workers/
├─ curation/
└─ runtime/
   ├─ repository/
   └─ retrieval/

server/guide-tools/
```

---

# 6. Workflow Integration

```text
Requirement
→ Guide Search
→ Design
→ Guide Reference
→ Implementation
→ Review against same Guide
```

Design, implementation, and review MUST use the same published Guide release.

Retrieval results must be bounded. Do not dump entire manuals into agent context.

Published guides are immutable for a release. Edit requires a new review/publish.

---

# 7. Required Tests

At minimum:

- Markdown ingest → sections → publish
- PDF/DOCX conversion produces normalized markdown (or explicit unsupported error)
- unpublished drafts are not retrievable at runtime
- release pin is stable across design and review
- metadata filter + keyword + FTS return expected sections
- mandatory/security rules outrank optional guides
- the sample rule “Controller에서 Repository 직접 호출 금지” is retrieved and used in design/review
- original files are stored via ArtifactStorage URIs, not copied into prompts
- vector DB is not introduced

---

# 8. Definition of Done

Phase 6 is complete only when a project guide such as:

```text
Controller에서 Repository 직접 호출 금지
```

causes agents to design:

```text
Controller → Service → Repository
```

and Review can verify the same rule.

Also required:

1. 6A ingestion pipeline and curation console exist.
2. Supported office/HWP/Markdown formats convert or fail explicitly.
3. Publish creates a pinned runtime release.
4. 6B retrieval is PostgreSQL FTS/trigram/metadata based.
5. Mandatory rules are enforced in search ranking.
6. Guide references are stored on the workflow.
7. No vector DB is used.

---

# 9. Implementation Instructions for Coding Agents

Before changing code:

1. Confirm Phase 5 golden workflows still pass.
2. Produce an implementation plan for 6A then 6B.
3. List files/modules to create or modify.
4. Identify any conflict with architecture rules.
5. Do not add a vector database.

During implementation:

- make small cohesive commits/changes
- add tests with each ingest/retrieve capability
- keep retrieval bounded

After implementation:

1. Run ingestion and retrieval tests.
2. Run the mandatory-rule design/review scenario.
3. Report architecture deviations.
4. Report remaining Phase 6 gaps.
5. Do not proceed to Phase 7 unless explicitly requested.
