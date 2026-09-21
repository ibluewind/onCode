# 로컬 실행 매뉴얼 (2026-09-17 기준)

Phase 3 슬라이스 1–4까지 구현된 상태를 전제로 한다. **마켓플레이스 배포가 아니며**, 추론은 실 vLLM이 아니라 서버 스크립트다. 코드 승인은 서버가 검증하고, **파일 반영은 Local Agent만** 한다.

```text
VS Code 확장  ←(localhost WebSocket+JSON)→  Local Agent  ←(gRPC 9443)→  Server  ← JDBC →  Postgres :5437
                         IDE는 서버에 직접 붙지 않는다.
```

Windows 경로·PowerShell 기준. 다른 OS는 설정 디렉터리만 다르다.

## 준비물

| 구성 | 버전 / 비고 |
|------|-------------|
| JDK | 25 LTS (`java -version`이 17이면 `JAVA_HOME`을 Temurin 25로) |
| Maven | 3.6.3+. 의존성은 저장소 `.mvn/settings.xml`의 Maven Central. **CodeGEN Nexus(8081) 쓰지 않음** |
| Go | 1.22+ (로컬은 1.27로 검증) |
| Node | 24 (확장 컴파일) |
| Docker | 로컬 Postgres만 |
| VS Code 또는 Cursor | 1.90+ 호환 확장. Marketplace 게시 없음 |

## 1. PostgreSQL

개발용 compose는 `127.0.0.1:5437`, DB/사용자 `oncode` / `oncode`. CodeGEN(5434)과 포트를 공유하지 않는다.

저장소 루트에서:

```powershell
docker compose -f infra/docker-compose.yml up -d
docker compose -f infra/docker-compose.yml ps
```

서버가 뜨면 Flyway가 스키마를 맞춘다. 통합 테스트는 이 compose를 쓰지 않고 Testcontainers를 쓴다.

## 2. Central Server

저장소 루트에서 HTTP **18080**, gRPC **9443**. 이 PC에서 **8080은 HRE Keycloak**이 쓰므로 쓰지 않는다.

```powershell
# JDK 25가 기본이 아니면 (예시)
$env:JAVA_HOME = "C:\Program Files\Eclipse Adoptium\jdk-25.0.4.101-hotspot"
$env:Path = "$env:JAVA_HOME\bin;" + $env:Path

mvn -s .mvn/settings.xml -f server/pom.xml spring-boot:run
```

확인: 브라우저 또는 `curl`로 `http://localhost:18080/actuator/health` → `{"status":"UP"}`.

gRPC는 HTTP가 아니다. 에이전트만 `127.0.0.1:9443`에 붙는다.

## 3. Local Agent

서버보다 **뒤에** 띄운다. `ONCODE_SERVER_GRPC_ADDR`가 비면 서버에 안 붙고, 채팅은 동작하지 않는다.

```powershell
$env:ONCODE_SERVER_GRPC_ADDR = "127.0.0.1:9443"
go run ./local-agent/cmd/oncode-agent -workspace "C:\Projects\onCode"
```

`-workspace`는 에이전트가 등록할 워크스페이스 루트다. actual diff 원문과 승인 후 apply가 이 루트만 읽거나 쓴다. 데모면 스크립트가 `src/AuthService.java`를 바꾸므로 **이 저장소가 아닌 임시/샘플 프로젝트**를 넣는 편이 안전하다.

정상 기동 시 stderr에 대략 다음이 나온다.

```text
server grpc connected 127.0.0.1:9443
ide ipc listening on 127.0.0.1:<port>/... (token in ...\oncode-agent)
```

Windows에서 IPC 디스커버리 파일:

```text
%APPDATA%\oncode-agent\ide-ipc.json
```

에이전트를 재시작하면 토큰이 바뀐다. 확장은 끊긴 뒤 이 파일을 다시 읽고 재연결한다.

선택 환경 변수:

| 변수 | 기본 | 의미 |
|------|------|------|
| `ONCODE_SERVER_GRPC_ADDR` | (비움) | 서버 gRPC `host:port` |
| `ONCODE_AGENT_STATE_DIR` | `%APPDATA%\oncode-agent` | `ide-ipc.json` 위치. 확장과 같아야 함 |
| `ONCODE_IDE_IPC_PORT` | `0` (임의 포트) | 고정 포트가 필요할 때만 |
| `ONCODE_IDE_IPC_BIND` | `127.0.0.1` | 루프백만 허용. `0.0.0.0`이면 기동 실패 |

## 4. VS Code / Cursor 확장

게시되지 않은 로컬 확장이다. 워크스페이스 파일을 읽거나 쓰지 않고, `ide-ipc.json`만 읽는다.

```powershell
cd ide\vscode
npm install
npm run compile
```

**개발 호스트 (F5)**  
`ide/vscode` 폴더를 연 뒤 `.vscode/launch.json`이 없으면 아래를 두고 F5한다.

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Run Extension",
      "type": "extensionHost",
      "request": "launch",
      "args": ["--extensionDevelopmentPath=${workspaceFolder}"]
    }
  ]
}
```

**VSIX 설치** (codegen 등 다른 폴더에서 쓸 때):

```powershell
cd ide\vscode
npm run package
```

생성된 `oncode-0.1.0.vsix`를 VS Code에서 **Install from VSIX**. 설치 후 창을 다시 로드한다 (`Developer: Reload Window`). 소스 고친 뒤에는 다시 패키지하고 VSIX를 재설치한다.

팔레트에 명령은 보이는데 `oncode.chat.send` not found 이면 확장이 활성화에 실패한 것이다. 예전 VSIX는 `ws`가 빠져 있었다. 위 `npm run package`로 만든 새 파일을 설치한다. Extension Host 로그는 Command Palette → **Developer: Show Logs...** → Extension Host.

에이전트와 확장의 state dir이 다르면 연결이 안 된다. 그 경우 설정 `oncode.agentStateDir`에 에이전트 state dir 절대 경로를 넣는다.

## 사용 순서

기동 순서: **Postgres → Server → Local Agent → 확장**. 확장만 먼저 켜도 재연결을 시도한다.

1. 편집기에서 **파일 하나를 연다.** (경로만 서버로 간다. 파일 본문은 보내지 않는다.)
2. Command Palette → **onCode: Send Chat**.
3. 요청 문구를 입력한다. 진행은 **Output → onCode**에 쌓인다.
4. 스크립트 Senior가 설계를 남기면 **설계 검토** 모달이 뜬다. **Approve** / **Reject**만 있다. Apply 없음.
5. 설계를 승인하면 구현·리뷰 후 Local Agent가 **등록된 `-workspace` 원문 대비 unified diff**를 만든다. **코드 diff** 모달에서 다시 Approve / Reject. Apply 버튼은 없다.
6. 거절하면 워크플로가 멈추고 파일은 그대로다. 코드 승인이 서버에서 통과하면 Local Agent가 `workspace.apply_changes`를 한 번 호출한다. 원문이 바뀌었으면 stale로 실패하고 강제 적용하지 않는다. apply 성공 후 서버는 `COMPLETED`로 닫고, 재연결 시 `APPLYING_CHANGE`를 반복하지 않는다. 제안이 현재 파일과 같으면 `unchanged`로 안내하고 디스크는 다시 쓰지 않는다.

파일이 없는 채팅은 서버가 현재 파일을 묻는다. InputBox에 경로를 답하면 된다.

재연결: **onCode: Reconnect to Local Agent**. 확장은 마지막 `work_item_id`를 확장 workspaceState에만 두고 `session.restore`로 복원한다.

## 지금 단계에서 기대하지 말 것

- 실 LLM / vLLM. 설계·제안 내용은 `ScriptedInferenceGateway` 고정 JSON이다. 에이전트가 붙어 있으면 설계 Approve 이후 `propose_changes`도 gRPC로 가므로 `src/AuthService.java`가 있어야 diff까지 진행된다.
- 빌드/테스트 실행 루프 (이벤트 표시만 일부 있음). IDE Apply 버튼.
- IDE ↔ 서버 직접 통신, REST로 채팅.
- 확장 Marketplace 검색·자동 업데이트.

## 자주 막히는 지점

| 증상 | 확인 |
|------|------|
| 서버 기동 실패, DB 연결 | compose `5437`이 떠 있는지. 5434(CodeGEN)에 붙지 않았는지 |
| Maven이 Nexus/8081로 감 | `-s .mvn/settings.xml` 또는 루트 `.mvn/maven.config` |
| `java -version`이 17 | `JAVA_HOME`을 JDK 25로 |
| `Application Control policy has blocked` / 애플리케이션 제어 정책 차단 | AppLocker 등이 `oncode-agent.exe`를 막음 (ISS-006). `go run`의 TEMP와 `bin\` 모두 해당할 수 있음. `secpol.msc` → AppLocker Executable Rules에 `C:\Projects\onCode\bin\*` Allow. `go build -o .\bin\oncode-agent.exe .\local-agent\cmd\oncode-agent` 후 그 exe로 기동 |
| 채팅이 안 감 | 에이전트에 `ONCODE_SERVER_GRPC_ADDR=127.0.0.1:9443`, 서버 9443 listen |
| 확장이 Local Agent를 못 찾음 | 에이전트 기동 여부, `%APPDATA%\oncode-agent\ide-ipc.json`, `oncode.agentStateDir` |
| `oncode.chat.send` not found | 확장이 로드되지 않음. `npm run package`로 만든 VSIX를 재설치한 뒤 Reload Window |
| Send Chat 후 모달이 없음 | Output `onCode`를 연다. `Local Agent IPC 미연결`이면 에이전트 기동 후 Reconnect. 파일 없이 보내면 경로를 묻는 InputBox가 먼저다 |
| 설계 모달만 반복 | 정상. 구현 루프는 설계 **Approve** 이후 |
| 승인했는데 파일이 안 바뀜 | `-workspace` 등록 여부, Output `apply.result` / stale 오류. 스크립트는 `src/AuthService.java`가 있어야 한다 |
| diff 승인 창이 안 닫힘 / 서버 UnexpectedRollback | 서버 재기동 + 새 VSIX 재설치 (ISS-004). 모달은 Approve/Reject만 있음 |
