# onCode Cursor / opencode Delivery Package

문서 전체 구조는 [`docs/README.md`](../README.md)를 본다.

## Overview files

1. `01_ARCHITECTURE.md` — compressed architecture overview
2. `02_IMPLEMENTATION_RULES.md` — non-negotiable implementation invariants
3. `03_ADR_SUMMARY.md` — ADR-001 ~ ADR-012 decisions
4. `04_IMPLEMENTATION_ROADMAP.md` — staged implementation plan

실행 태스크는 overview에 두지 않는다. Phase별 명세는 [`../tasks/`](../tasks/)다.

| Phase | Task |
|-------|------|
| 0 Protocol Foundation | [`../tasks/PHASE_00.md`](../tasks/PHASE_00.md) |
| 1 Local Agent Core | [`../tasks/PHASE_01.md`](../tasks/PHASE_01.md) |
| 2 Server Workflow Core | [`../tasks/PHASE_02.md`](../tasks/PHASE_02.md) |
| 3 IDE Interaction | [`../tasks/PHASE_03.md`](../tasks/PHASE_03.md) |
| 4 Project Intelligence | [`../tasks/PHASE_04.md`](../tasks/PHASE_04.md) |
| 5 E2E Implementation Loop | [`../tasks/PHASE_05.md`](../tasks/PHASE_05.md) |
| 6 Guide | [`../tasks/PHASE_06.md`](../tasks/PHASE_06.md) |
| 7 Dependency / Security | [`../tasks/PHASE_07.md`](../tasks/PHASE_07.md) |
| 8 Auth / Operations / Hardening | [`../tasks/PHASE_08.md`](../tasks/PHASE_08.md) |

Recommended coding-agent instruction:

```text
Read docs/overview/01_ARCHITECTURE.md, docs/overview/02_IMPLEMENTATION_RULES.md,
docs/overview/03_ADR_SUMMARY.md, docs/overview/04_IMPLEMENTATION_ROADMAP.md
and the task document for the assigned phase under docs/tasks/.

Implement only the assigned phase.
Before writing code, inspect the repository and produce an implementation plan.
Do not violate docs/overview/02_IMPLEMENTATION_RULES.md.
Do not proceed to later phases unless explicitly instructed.
```
