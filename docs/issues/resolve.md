# Resolved Issues

해결된 이슈만 이 파일에 둔다. 최신이 위로 오게 추가한다.

---

## ISS-002 — capabilities result에 PHASE_01 필드 미반영

- 발견일: 2026-09-14
- 해결일: 2026-09-14
- Phase: 1
- 원인: Phase 0 capabilities 계약이 `agent_version`/`platform`/`capabilities`만 정의했고, PHASE_01의 `protocol_version`·`supported_tools`는 Local Agent에만 먼저 들어갔다.
- 조치: JSON Schema·proto·Go DTO·fixture·계약 테스트에 필드를 추가(proto 필드 번호 4, 5). Local Agent는 `messages.CapabilitiesResult`를 직접 사용하도록 정리.
- 관련 커밋 / 파일: `protocol/schemas/json/capabilities.schema.json`, `protocol/schemas/proto/oncode_tool.proto`, `protocol/messages/messages.go`, `protocol/tests/fixtures/capabilities.json`, `protocol/contract/contract_test.go`, `local-agent/internal/tools/system/system.go`
- 재발 방지: PHASE_01 tool result 필드는 `protocol/` 계약과 동시에 갱신. Local Agent 전용 확장 DTO를 두지 않는다.

---

## ISS-001 — Go 툴체인 없음

- 발견일: 2026-09-09
- 해결일: 2026-09-09
- Phase: 0
- 원인: 저장소에 Go 모듈을 추가했으나 개발 머신 PATH에 `go`가 없었다.
- 조치: `winget install --id GoLang.Go`로 Go 1.27.0 windows/amd64 설치. `go test ./protocol/...` 통과.
- 관련 커밋 / 파일: `go.mod`, `protocol/**`
- 재발 방지: Phase 1 Local Agent 작업 전에 `go version` 확인. 신규 환경은 Go 1.22+를 전제로 한다.

---

<!--
## ISS-000 — 제목

- 발견일:
- 해결일:
- Phase:
- 원인:
- 조치:
- 관련 커밋 / 파일:
- 재발 방지:
-->
