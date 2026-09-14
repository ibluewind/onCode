# onCode Documentation

문서 루트는 역할별로 나눈다.

| Folder | Purpose |
|--------|---------|
| [`overview/`](overview/README.md) | 코딩 에이전트용 압축 아키텍처 패키지 |
| [`spec/`](spec/) | 상세 스펙 SPEC-01 ~ SPEC-15 |
| [`adr/`](adr/) | Architecture Decision Records ADR-001 ~ ADR-012 |
| [`tasks/`](tasks/) | 단계별 실행 태스크 명세 |
| [`plans/`](plans/) | 일별 작업 계획 `YYYY-MM-DD-작업계획.md` |
| [`issues/`](issues/) | 이슈 `open.md` / `resolve.md` |
| [`logs/`](logs/) | 일별 작업 로그 `YYYY-MM-DD.md` |
| [`HANDOFF.md`](HANDOFF.md) | 다음 세션 인수인계 |

## Implementation tasks

로드맵 Phase 0~8과 1:1로 대응한다. 상세는 [`overview/04_IMPLEMENTATION_ROADMAP.md`](overview/04_IMPLEMENTATION_ROADMAP.md).

| Task | Title |
|------|-------|
| [`tasks/PHASE_00.md`](tasks/PHASE_00.md) | Protocol Foundation |
| [`tasks/PHASE_01.md`](tasks/PHASE_01.md) | Local Agent Core |
| [`tasks/PHASE_02.md`](tasks/PHASE_02.md) | Server Workflow Core |
| [`tasks/PHASE_03.md`](tasks/PHASE_03.md) | IDE Interaction |
| [`tasks/PHASE_04.md`](tasks/PHASE_04.md) | Project Intelligence |
| [`tasks/PHASE_05.md`](tasks/PHASE_05.md) | End-to-End Implementation Loop (Technical MVP) |
| [`tasks/PHASE_06.md`](tasks/PHASE_06.md) | Guide |
| [`tasks/PHASE_07.md`](tasks/PHASE_07.md) | Dependency / Security |
| [`tasks/PHASE_08.md`](tasks/PHASE_08.md) | Authentication / Operations / Hardening |

구현을 시작할 때는 [`overview/README.md`](overview/README.md)의 권장 지시문을 따르고, **명시된 Phase만** 구현한다.
