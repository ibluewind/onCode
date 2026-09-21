# Resolved Issues

해결된 이슈만 이 파일에 둔다. 최신이 위로 오게 추가한다.

---

## ISS-006 — Local Agent Application Control 차단

- 발견일: 2026-09-21
- 해결일: 2026-09-21
- Phase: 3 (로컬 실행 환경)
- 원인: Windows Application Control(AppLocker 등)이 미서명 `oncode-agent.exe` 실행을 거부했다. `go run`의 `%TEMP%\go-build…\exe\`뿐 아니라 `C:\Projects\onCode\bin\oncode-agent.exe`도 동일 메시지로 차단됐다. onCode 코드·빌드 오류가 아니다.
- 조치: 로컬 `secpol.msc` → 애플리케이션 제어 정책 → AppLocker → Executable Rules에 `C:\Projects\onCode\bin\*` (또는 해당 exe) Allow를 추가하고 규칙을 적용했다.
- 관련 커밋 / 파일: 코드 변경 없음. 실행: `go build -o .\bin\oncode-agent.exe .\local-agent\cmd\oncode-agent` 후 `.\bin\oncode-agent.exe -workspace …`
- 재발 방지: 새 머신·새 경로에서 같은 오류(`An Application Control policy has blocked this file` / `애플리케이션 제어 정책에서 이 파일을 차단`)면 AppLocker/WDAC/Smart App Control 로그를 보고 허용 경로를 추가한다. 회사 PC는 IT 허용 목록이 필요할 수 있다. `%TEMP%`만 열어 두면 `go run`은 되지만, 프로젝트 `bin\` 허용이 더 안정적이다.

---

## ISS-005 — design Approve 후 propose_changes TIMEOUT

- 발견일: 2026-09-21
- 해결일: 2026-09-21
- Phase: 3
- 원인: `approval.decide`가 gRPC `onNext` 수신 스레드에서 동기 처리되면서 `propose_changes` RESPONSE를 같은 스레드에서 기다렸다. RESPONSE를 받을 스레드가 막혀 5초 후 TIMEOUT. 추가로 `a.Serve`가 이미 열린 스트림에 Connect를 다시 호출할 수 있었다.
- 조치: RESPONSE `complete`는 수신 스레드에서 즉시 처리. EVENT는 워커 풀로 넘김. 송신은 `GrpcToolSession.send`로 직렬화. 에이전트는 `transport.Loop`만 사용.
- 관련 커밋 / 파일: `AgentSessionService.java`, `GrpcToolSession.java`, `local-agent/cmd/oncode-agent/main.go`
- 재발 방지: 같은 bidi 스트림에서 도구 응답을 기다리는 도메인 처리를 수신 콜백 안에서 하지 않는다.

---

## ISS-004 — bindDiff UnexpectedRollback / diff 모달

- 발견일: 2026-09-17
- 해결일: 2026-09-17
- Phase: 3
- 원인: `bindDiff`가 `@Transactional requireUsable` 예외를 잡아 Rejected로 바꾸면 트랜잭션이 rollback-only인 채로 commit한다. 예외가 gRPC `onNext`로 새면 스트림이 끊기고, 복원 이벤트가 같은 승인 창을 다시 띄운다. VS Code 모달은 X 버튼이 없다.
- 조치: bind는 `findById`만 사용. `requireUsable`에 `noRollbackFor`. 컨텍스트 부재는 `find`. gRPC 예외는 error 이벤트. IDE는 approval_id 중복 모달 억제.
- 관련 커밋 / 파일: `ApprovalIntake.java`, `ApprovalService.java`, `ContextService.java`, `AgentSessionService.java`, `ide/vscode/extension/activate.ts`
- 재발 방지: 트랜잭션 메서드 안에서 다른 `@Transactional` 조회 예외를 잡아 정상 반환하지 않는다. 없으면 Optional.

---

## ISS-003 — VS Code `oncode.chat.send` not found

- 발견일: 2026-09-17
- 해결일: 2026-09-17
- Phase: 3
- 원인: `contributes.commands`만 있으면 팔레트에 명령이 보인다. `activate`가 `ws`를 require하는데 기존 VSIX에 `node_modules/ws`가 없어 모듈 로드가 실패하고 핸들러가 등록되지 않았다.
- 조치: esbuild로 `out/main.js`에 `ws`를 묶고, VSIX에는 `package.json`과 그 파일만 넣는다.
- 관련 커밋 / 파일: `ide/vscode/package.json`, `ide/vscode/.vscodeignore`, `ide/vscode/oncode-0.1.0.vsix`, `docs/LOCAL_RUN.md`
- 재발 방지: `npm run package`로만 VSIX를 만들고, 설치 후 Reload Window. 팔레트만 보이고 명령이 없으면 Extension Host 로그를 본다.

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
