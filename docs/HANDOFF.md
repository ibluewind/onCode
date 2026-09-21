# HANDOFF

다음 세션은 이 파일부터 읽는다. 직전 일자 로그: `docs/logs/2026-09-21.md`.

## 현재 위치

- Phase: **3 — IDE Interaction** (`docs/tasks/PHASE_03.md`)
- 상태: **DoD 기준 완료** (슬라이스 1–6 + apply/gRPC/COMPLETED 보강). Phase 4 미착수.
- 마지막 자동 검증 (2026-09-21):
  - Go `idesession` **14 PASS** (typed-nil Listen 수정 포함)
  - Maven **Tests run: 50**, BUILD SUCCESS (16:15 KST)
  - `ide/vscode` npm **13 pass / 0 fail**
- 계획서: `docs/plans/2026-09-21-작업계획.md`
- 구현결과: `docs/results/2026-09-21-구현결과.md`
- 하루 종료: 2026-09-21 16:46 KST. 커밋하지 않음.

## 직전 작업 요약

1. IDE ↔ Agent WS+JSON, Agent ↔ Server gRPC Envelope
2. 설계/코드 승인, Local Agent actual diff, 재연결 복원
3. `-workspace` 원문 + 승인 후 apply, 도구 채널 gRPC
4. `apply.completed` → `COMPLETED`, unchanged 안내
5. ISS-005 TIMEOUT, ISS-006 AppLocker, typed-nil Stream 패닉 수정

## 다음에 할 한 가지

지시가 오면 계획서부터 쓴다. 후보(자동 진행 금지):

- (선택) `ONCODE_SERVER_GRPC_ADDR=127.0.0.1:9443` + 에이전트 재기동 후 COMPLETED/unchanged 수동 E2E, 필요 시 VSIX 재패키지
- 지시가 있으면 Phase 4 Project Intelligence (`docs/tasks/PHASE_04.md`)

## 열어 둔 이슈

없음. 해결: ISS-003–ISS-006 (`docs/issues/resolve.md`).

## 읽어야 할 파일

- `docs/logs/2026-09-21.md`
- `docs/results/2026-09-21-구현결과.md`
- `docs/LOCAL_RUN.md`
- `docs/tasks/PHASE_04.md` (Phase 4를 지시받은 경우만)

## 하지 말 것

- 계획서 없이 Phase 4 / 실 vLLM / hunk approval
- 서버가 디스크에 쓰기, 승인만으로 강제 apply
- V1/V2 DDL 수정, git commit/push를 Agent가 임의 추가
- 로컬 CodeGEN Nexus로 onCode 의존성 해석, `~/.m2/settings.xml` 수정
- IDE→서버 직접 호출

## 환경 참고

- JDK 25 + `mvn -s .mvn/settings.xml`
- 개발 Postgres: `127.0.0.1:5437` / 테스트는 Testcontainers 격리
- 서버: HTTP `18080`, gRPC `127.0.0.1:9443`
- Local Agent: 사용자/셸 `ONCODE_SERVER_GRPC_ADDR`, `-workspace`, IPC `%APPDATA%\oncode-agent`
- AppLocker: `C:\Projects\onCode\bin\*` Allow (ISS-006)
- 추론: `ScriptedInferenceGateway` (실 vLLM 없음)
