# HANDOFF

다음 세션은 이 파일부터 읽는다. 직전 일자 로그: `docs/logs/2026-09-09.md`.

## 현재 위치

- Phase: 0 — Protocol Foundation (`docs/tasks/PHASE_00.md`)
- 상태: `protocol/` 계약이 들어가고 `go test ./protocol/...` 통과. protobuf 코드젠(`protoc`)은 하지 않음.
- 2026-09-09 작업은 종료됨.

## 직전 작업 요약

- 문서 구조(overview / spec / adr / tasks)와 Phase 0~8 태스크 문서를 정리함.
- 작업 규칙: 계획서, 이슈, 일일 로그, HANDOFF.
- Tool Protocol v1: ID, 에러, JSON Schema, `.proto`, 계약 테스트.
- 프로토콜 문자열: `oncode-tool/1.0`
- ID: `{PREFIX}-{UUIDv7}`
- JSON `payload` ↔ proto typed 필드 + `*_json` 어댑터.

## 다음에 할 한 가지

사용자 지시를 기다린다.

- Phase 0 잔여를 닫으려면: `protoc`로 Go/Java stub 생성 (현재는 필수 아님).
- Phase 1을 시작하면: 그날 계획서를 쓰고 `docs/tasks/PHASE_01.md`만 구현한다.

## 열어 둔 이슈

없음. ISS-001은 `docs/issues/resolve.md`.

## 읽어야 할 파일

- `docs/logs/2026-09-09.md`
- `protocol/README.md`
- `docs/tasks/PHASE_00.md`
- `docs/tasks/PHASE_01.md` (Phase 1 시작 시)
- `docs/issues/open.md`

## 하지 말 것

- 계획서 없이 코드 변경
- Local Agent / Server / IDE를 Phase 0 잔여로 확장
- protobuf JSON 이름을 camelCase로 변경 (`json_name`이 논리 프로토콜)
