# Resolved Issues

해결된 이슈만 이 파일에 둔다. 최신이 위로 오게 추가한다.

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
