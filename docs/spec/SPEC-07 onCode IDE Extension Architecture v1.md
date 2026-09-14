# SPEC-07 onCode IDE Extension Architecture v1

## 1. 목적

본 명세는 onCode IDE Extension의 아키텍처를 정의한다.

IDE Extension은 개발자가 onCode와 상호작용하는 사용자 인터페이스 계층이며, 실제 파일 작업, 빌드, 테스트, Git 실행, 프로젝트 분석은 Local Agent에 위임한다.

지원 IDE:

- Eclipse
- Visual Studio Code
- IntelliJ IDEA

핵심 원칙은 다음과 같다.

> IDE Extension은 Thin Client로 유지하고, 실행 로직과 프로젝트 처리 로직은 Local Agent에 집중한다.

---

# 2. 전체 구조

```text
┌──────────────────────────── IDE ────────────────────────────┐
│                                                           │
│  Chat View                                                │
│  Progress View                                            │
│  Question / Selection UI                                  │
│  Design Review UI                                         │
│  Diff Review UI                                           │
│  Approval UI                                              │
│  Notification                                             │
│                                                           │
│  Editor Context Collector                                 │
│  Command Adapter                                          │
│                                                           │
└────────────────────────────┬───────────────────────────────┘
                             │
                      IDE Bridge Protocol
                             │
                             ▼
┌──────────────────────── Local Agent ────────────────────────┐
│                                                           │
│ IDE Bridge                                                │
│ Local Agent Core                                          │
│ Project Intelligence                                      │
│ Workspace Manager                                         │
│ Execution Runtime                                         │
│ Policy / Approval Engine                                  │
│ Server Connector                                          │
│                                                           │
└───────────────────────────────────────────────────────────┘
```

---

# 3. 책임 범위

IDE Extension이 담당하는 기능:

```text
Chat UI

Editor Context 수집

Selection Context 수집

Progress 표시

질문 / 선택지 표시

설계 승인 UI

Diff 표시

파일별 변경 확인

승인 / 반려 입력

Git / Build / Test 명령 요청 UI

Notification

Local Agent 상태 표시
```

IDE Extension이 담당하지 않는 기능:

```text
Project Scan

AST Parsing

PROJECT_INDEX 생성

Source 직접 분석

Build 실행

Test 실행

Git 실행

Shell 실행

Nexus 검색

Security 검사

Workflow 상태 결정
```

---

# 4. IDE Extension 공통 구조

세 IDE에서 동일한 논리 컴포넌트를 유지한다.

```text
IDE Extension

├─ Chat UI
├─ Context Collector
├─ Progress Controller
├─ Review Controller
├─ Approval Controller
├─ Command Controller
├─ Local Agent Client
└─ IDE Adapter
```

IDE-specific API는 `IDE Adapter`에 격리한다.

---

# 5. IDE Adapter

IDE별 API 차이를 추상화한다.

공통 인터페이스 예:

```text
IDEAdapter

getCurrentFile()

getCurrentSelection()

getCursorPosition()

getOpenFiles()

openFile(path)

revealRange(path, startLine, endLine)

showDiff(changeSet)

showQuestion(question)

showApproval(request)

showProgress(event)

showNotification(message)

getDiagnostics()
```

---

# 6. IDE별 구현

```text
IDEAdapter
 ├─ EclipseAdapter
 ├─ VSCodeAdapter
 └─ IntelliJAdapter
```

이 구조를 유지하여 Chat/Approval 등의 상위 로직을 재사용한다.

---

# 7. Local Agent 연결

IDE Extension은 중앙 서버에 직접 연결하지 않는 것을 기본 원칙으로 한다.

```text
IDE Extension
      ↓
Local Agent
      ↓
onCode Server
```

장점:

```text
IDE별 Server Protocol 구현 최소화

인증/세션 처리 일원화

Local 상태 처리 일원화

서버 연결 장애 처리 일원화

IDE Plugin 복잡도 감소
```

---

# 8. IDE ↔ Local Agent Protocol

Local IPC를 사용한다.

후보:

```text
WebSocket

Local HTTP

gRPC

Named Pipe

Unix Domain Socket
```

권장 초기 구현:

```text
localhost WebSocket
+
JSON message
```

또는 Local Agent 구현 언어/환경에 따라 gRPC를 사용할 수 있다.

---

# 9. Localhost 보안

IDE Bridge 서버는 기본적으로 다음과 같이 제한한다.

```text
127.0.0.1 only

외부 Interface bind 금지

Random session token

IDE Client Registration

Short-lived connection token
```

임의 로컬 프로세스가 Local Agent를 호출하지 못하도록 인증을 적용한다.

---

# 10. IDE 등록

IDE Extension 시작 시 Local Agent에 등록한다.

예:

```json
{
  "type": "ide.register",

  "data": {
    "ide": "INTELLIJ",
    "extension_version": "1.0.0",
    "workspace": "current"
  }
}
```

Local Agent는 IDE Client ID를 반환한다.

---

# 11. 다중 IDE 처리

동일 프로젝트를 여러 IDE에서 열 수도 있다.

예:

```text
VS Code
   │
IntelliJ
   │
   └── Local Agent
```

Local Agent는 IDE Client별 세션을 관리한다.

다만 기본 활성 IDE는 현재 사용자 요청을 보낸 Client로 한다.

---

# 12. Chat UI

Chat은 onCode의 주 인터페이스다.

지원 입력:

```text
자연어 요청

코드 구현 요청

수정 요청

검토 요청

검색 요청

테스트 요청

빌드 요청

Git 요청

일반 질문
```

---

# 13. Chat Request Context

사용자의 메시지와 함께 최소 Context를 전달한다.

예:

```json
{
  "message": "이 메서드에 null 처리를 추가해줘.",

  "ide_context": {
    "current_file": "src/main/java/UserService.java",

    "selection": {
      "start_line": 42,
      "end_line": 67
    },

    "cursor": {
      "line": 51,
      "column": 12
    }
  }
}
```

---

# 14. Source 자동 첨부 금지

현재 파일 전체 Source나 열린 모든 파일을 자동으로 중앙 서버에 전송하지 않는다.

IDE는 식별 정보만 전달한다.

```text
current file path

selection range

symbol name
```

실제 Source는 Server가 Local Agent Tool을 통해 필요할 때 요청한다.

---

# 15. Selected Text

사용자가 특정 코드를 선택한 상태에서 질문하면 Selection 내용을 Local Agent가 전달할 수 있다.

정책에 따라 다음 두 방식 중 하나를 사용할 수 있다.

```text
A. Selection source 포함

B. path + range만 전달
```

보안과 일관성을 위해 기본은 B를 권장한다.

Local Agent가 현재 파일 hash와 함께 실제 내용을 제공한다.

---

# 16. Slash Command / IDE Command

Chat 외에도 명시적인 명령을 제공할 수 있다.

예:

```text
/oncode.review

/oncode.test

/oncode.build

/oncode.git.status

/oncode.explain

/oncode.fix
```

하지만 UI에서 Slash Command를 강제할 필요는 없다.

자연어 Intent Classification이 기본이다.

---

# 17. Context Menu

Editor Context Menu 예:

```text
onCode로 설명

onCode로 검토

onCode로 수정

테스트 생성

이 오류 분석

관련 코드 찾기
```

선택한 코드/파일 Context와 함께 요청을 전송한다.

---

# 18. Code Action

IDE가 지원하는 경우 Diagnostic 기반 Code Action도 제공할 수 있다.

예:

```text
Cannot resolve symbol
```

옆에:

```text
Ask onCode
```

Action 표시.

---

# 19. Diagnostics 전달

IDE 자체 진단 결과를 전달할 수 있다.

```json
{
  "diagnostics": [
    {
      "file": "UserService.java",
      "line": 42,
      "severity": "ERROR",
      "message": "Cannot resolve method..."
    }
  ]
}
```

다만 IDE Diagnostics는 보조 정보이며, 실제 Build/Test 결과와 구분한다.

---

# 20. Progress UI

Server Workflow 상태를 실시간 표시한다.

예:

```text
✓ 요청 분석

✓ 관련 코드 탐색

✓ 개발 가이드 확인

● 구현 설계

○ 설계 확인

○ 코드 작성

○ 코드 검토

○ 변경 승인

○ 빌드

○ 테스트
```

---

# 21. Progress Event

Local Agent로부터 다음 Event를 받는다.

```json
{
  "event": "workflow.progress",

  "data": {
    "stage": "DESIGNING",
    "status": "RUNNING",
    "message": "기능 구현 방식을 설계하고 있습니다."
  }
}
```

IDE는 내부 State 이름보다 사용자 친화적 메시지를 표시한다.

---

# 22. Progress Details

기본 화면은 단순하게 유지하되 상세보기를 제공할 수 있다.

예:

```text
개발 가이드 검색

- Spring Security 인증 정책
- 조직 보안 코딩 규칙
- 프로젝트 로그인 구현 규칙
```

Agent 내부 reasoning은 표시하지 않는다.

---

# 23. Question UI

Senior Developer가 추가 정보를 요청할 경우 IDE에서 질문을 표시한다.

예:

```text
계정 잠금 해제 방식을 선택해 주세요.

○ 관리자 직접 해제
○ 30분 후 자동 해제
○ 기존 프로젝트 정책 유지
```

---

# 24. Question Type

지원 타입:

```text
SINGLE_SELECT

MULTI_SELECT

TEXT

CONFIRM
```

초기에는 `SINGLE_SELECT`, `TEXT`, `CONFIRM` 정도면 충분하다.

---

# 25. 선택지 우선

Senior Developer의 질문은 가능한 경우 선택지를 포함하도록 한다.

이유:

```text
사용자 판단 시간 단축

모호한 답변 감소

USER_DECISION 구조화 용이

Workflow 자동 재개 용이
```

---

# 26. Free Text Option

선택지 이외의 답변도 허용할 수 있다.

예:

```text
○ A
○ B
○ C
○ 직접 입력
```

---

# 27. Design Review UI

구현 전 설계를 개발자에게 표시한다.

최소 표시 항목:

```text
작업 목표

변경 예정 파일

추가 파일

주요 구현 로직

Dependency 변경

보안 고려

테스트 계획

적용 개발 가이드
```

---

# 28. Design Review 예

```text
구현 계획

목표
로그인 실패 5회 이상 시 계정 잠금

변경 파일
- AuthService.java
- User.java
- UserRepository.java

구현 방식
- 로그인 실패 시 failureCount +1
- 5 이상인 경우 LOCKED 상태 저장
- 정상 로그인 시 failureCount 초기화

테스트
- 4회 실패 시 ACTIVE
- 5회 실패 시 LOCKED
- 성공 시 count 초기화
```

---

# 29. Design Approval Action

```text
승인

수정 요청

반려

취소
```

수정 요청에는 Reason 입력을 권장한다.

---

# 30. Design 반려 사유

예:

```text
DB 필드는 추가하지 말고 기존 상태 컬럼을 사용해줘.
```

IDE는 이 값을 Server로 전달하고 Workflow는 설계 단계로 복귀한다.

---

# 31. Diff Review UI

코드 변경 전 반드시 Diff를 표시한다.

```text
Current Workspace
      ↓
Local Agent Diff
      ↓
IDE Diff Viewer
```

Server가 만든 Diff 텍스트 자체보다 Local Agent가 현재 파일을 기준으로 생성한 Diff를 신뢰한다.

---

# 32. IDE Native Diff 우선

가능하면 IDE가 제공하는 Native Diff Viewer를 사용한다.

장점:

```text
익숙한 UI

Syntax Highlight

Side-by-side

Line Navigation

File Navigation
```

---

# 33. Diff 표시 정보

파일별로:

```text
CREATE

MODIFY

DELETE

RENAME
```

를 명확하게 표시한다.

예:

```text
M AuthService.java

+ AccountLockPolicy.java

D LegacyAuthUtil.java
```

---

# 34. Diff 승인

기본 정책:

```text
Change Set 전체 승인

Change Set 전체 반려
```

MVP에서는 Atomic Approval을 권장한다.

---

# 35. 파일별 승인

향후 지원 가능:

```text
Accept File

Reject File
```

하지만 파일 간 의존성 문제가 있으므로 서버에서 partial approval 가능 여부를 판단해야 한다.

---

# 36. Inline Approval

더 발전된 UX에서는 파일 안의 개별 hunk 승인도 가능하다.

그러나 v1에서는 제외하는 것을 권장한다.

이유:

```text
Change Set 일관성 복잡도 증가

Design과 실제 적용 결과 불일치 가능

Server 재검증 필요
```

---

# 37. Code Reject

코드 반려 시 reason을 입력한다.

예:

```text
"기존 Utility를 사용하지 않고 새 클래스를 만든 이유가 납득되지 않음."
```

Workflow는 설계 또는 분석 단계로 돌아간다.

---

# 38. Stale Diff

Diff 표시 중 개발자가 파일을 직접 변경할 수 있다.

Local Agent가 이를 감지하면 IDE에:

```text
변경 대상 파일이 수정되었습니다.
기존 변경안은 더 이상 적용할 수 없습니다.
```

를 표시한다.

상태:

```text
STALE_WORKSPACE
```

---

# 39. Stale Diff Action

사용자에게 다음 Action을 제공한다.

```text
재분석

변경안 폐기
```

기존 Diff를 강제 적용하는 옵션은 기본 제공하지 않는다.

---

# 40. Chat Message Type

Chat Stream은 단순 assistant text 외에도 구조화된 메시지를 지원한다.

예:

```text
TEXT

PROGRESS

QUESTION

DESIGN

DIFF

APPROVAL

RESULT

WARNING

ERROR
```

---

# 41. Message Card

UI는 Message Type별 Card를 사용할 수 있다.

예:

```text
[설계 검토]

[질문]

[코드 변경]

[빌드 결과]

[테스트 결과]
```

Chat 안에 Workflow Action이 자연스럽게 이어지도록 한다.

---

# 42. Build Result UI

Build 성공:

```text
Build succeeded

Maven
Duration: 12.4 sec
Warnings: 2
```

실패:

```text
Build failed

AuthService.java:84
cannot find symbol: UserStatus.LOCKED
```

---

# 43. 오류 위치 이동

Build/Test/Review Finding에 File/Line이 존재하면 클릭해서 해당 위치로 이동한다.

```text
AuthService.java:84
```

→ Editor open/reveal.

---

# 44. Test Result UI

예:

```text
Tests

32 passed
1 failed
2 skipped
```

실패 Test:

```text
AuthServiceTest.loginShouldLockAccount

Expected LOCKED
Actual ACTIVE
```

클릭 시 Test File로 이동한다.

---

# 45. Review Finding UI

코드 리뷰 결과도 IDE 문제 패널 또는 Chat Card로 표현할 수 있다.

예:

```text
HIGH

AuthService.java:82

개발 가이드 GUIDE-87 위반
로그인 실패 시 audit logging이 필요합니다.
```

---

# 46. Security Finding UI

Security Finding은 일반 Review보다 명확하게 구분한다.

예:

```text
SECURITY / HIGH

BuildService.java:42

사용자 입력이 Shell 명령에 직접 포함됩니다.
```

---

# 47. Git UI

사용자가 Git 작업을 요청하면 실제 명령보다 작업 의미를 먼저 보여주는 것이 좋다.

예:

```text
Commit 예정

Branch: feature/login
Files: 3

Message:
feat: add account lock policy
```

[승인] [취소]

---

# 48. High-risk Git UI

예:

```text
주의

이 작업은 원격 저장소에 변경을 전송합니다.

git push origin feature/login
```

[Push 승인] [취소]

---

# 49. Destructive Operation UI

파괴적 작업은 별도 경고 UI를 사용한다.

예:

```text
위험한 작업

git reset --hard HEAD~1

현재 커밋되지 않은 변경사항이 삭제될 수 있습니다.
```

사용자 명시 승인 없이는 실행하지 않는다.

---

# 50. Shell Command UI

Shell 요청이 필요한 경우 실행 명령을 그대로 표시한다.

```text
실행 예정 명령

mvn clean package -DskipTests
```

다만 Build/Test/Git 전용 Tool이 있다면 Shell UI를 사용하지 않는다.

---

# 51. Cancel

사용자는 진행 중인 Workflow를 취소할 수 있다.

UI:

```text
[작업 중지]
```

IDE:

```text
Local Agent
→ Server
→ Workflow CANCEL
```

실행 중 Build/Test가 있으면 Process Cancellation을 시도한다.

---

# 52. 작업 중지와 Undo 구분

`Cancel`은 이후 Workflow를 중지한다.

이미 승인되어 적용된 Source 변경을 자동으로 되돌린다는 의미는 아니다.

Rollback은 별도 기능이다.

---

# 53. Apply 후 Undo

onCode가 적용한 Change Set은 Local Agent Backup을 이용하여 Undo 기능을 제공할 수 있다.

예:

```text
onCode 변경 되돌리기
```

단 다음 상황에서는 주의:

```text
적용 후 개발자가 추가 수정

다른 Change Set 적용

Git commit 완료
```

v1에서는 단순 직전 Change Set Undo 정도만 고려한다.

---

# 54. Session UI

IDE 상태 표시:

```text
onCode: Connected

onCode: Reconnecting

onCode: Local Agent unavailable

onCode: Server unavailable
```

---

# 55. Local Agent 미실행

Extension 실행 시 Local Agent 연결에 실패하면:

```text
onCode Local Agent가 실행되고 있지 않습니다.
```

를 표시한다.

제품 정책에 따라 IDE에서 Local Agent 실행을 요청할 수 있다.

---

# 56. Server Disconnect

Local Agent는 연결을 유지하려 시도하고 IDE에는:

```text
중앙 서버 연결이 끊어졌습니다.
재연결을 시도하고 있습니다.
```

와 같이 상태를 표시한다.

---

# 57. Offline Behavior

중앙 Server가 없으면 Agentic Coding 기능은 제한된다.

가능한 Local-only 기능은 향후 정의할 수 있다.

예:

```text
project summary 조회

git.status

local diff
```

다만 v1에서는 Server 연결 필수를 단순한 정책으로 둘 수 있다.

---

# 58. Authentication

사용자 로그인 UI는 IDE Extension에서 제공할 수 있다.

흐름:

```text
IDE
 ↓
Local Agent
 ↓
Server Authentication
```

Token은 가능하면 IDE Plugin보다 Local Agent가 관리한다.

---

# 59. Token 저장

IDE별 Credential Storage에 분산 저장하기보다 Local Agent Secure Store로 집중하는 것을 권장한다.

IDE Extension은 Session ID 정도만 유지한다.

---

# 60. Workspace Registration

IDE가 프로젝트를 열면 Local Agent에 현재 Workspace 정보를 전달한다.

```json
{
  "event": "workspace.opened",

  "data": {
    "root": "...",
    "ide": "VSCODE"
  }
}
```

Local Agent가 실제 Project Fingerprint 및 Registration을 처리한다.

---

# 61. Workspace Close

```text
workspace.closed
```

Event를 전달할 수 있다.

하지만 다른 IDE Client가 같은 Workspace를 사용 중이면 Local Agent Project는 유지한다.

---

# 62. Current File Event

사용자가 Editor File을 변경할 때마다 Server에 전달할 필요는 없다.

IDE → Local Agent 내부 상태만 갱신한다.

실제 사용자 요청 시 현재 Context를 포함한다.

---

# 63. Cursor Event

Cursor 이동을 계속 Stream하지 않는다.

요청 시점에만 Snapshot한다.

불필요한 Traffic을 줄인다.

---

# 64. Open Files

열려 있는 파일 목록은 보조 Context로 사용할 수 있다.

하지만 이를 relevance 판단의 주요 기준으로 삼지 않는다.

Project Search가 더 우선이다.

---

# 65. Terminal 통합

onCode 전용 Terminal UI를 별도로 만들 필요는 없다.

Build/Test/Shell 결과는 기존 IDE Terminal 또는 onCode Output Panel에 표시할 수 있다.

초기에는 onCode 전용 Output Panel을 권장한다.

---

# 66. Output Channel

예:

```text
onCode

onCode Build

onCode Test

onCode Git
```

등을 별도 Channel로 나눌 수 있다.

---

# 67. Build Full Log

Chat에는 Summary만 표시한다.

전체 로그는:

```text
[전체 로그 보기]
```

를 통해 Output Panel에서 보여준다.

---

# 68. Logging 민감정보

IDE 표시 로그도 Secret Masking을 적용한 결과를 사용하는 것이 좋다.

Server/Local Agent가 이미 마스킹한 데이터를 사용한다.

---

# 69. Notification

장시간 작업 완료:

```text
onCode 작업이 완료되었습니다.
```

IDE Notification을 표시할 수 있다.

하지만 불필요한 Pop-up을 최소화한다.

---

# 70. Background Workflow

사용자가 다른 파일을 편집하는 동안 Workflow가 진행될 수 있다.

IDE는 특정 Editor를 강제로 고정하지 않는다.

---

# 71. 여러 Work Item

Chat Session 안에서 여러 요청이 연속될 수 있다.

각 Work Item을 내부적으로 구분한다.

UI에서는:

```text
로그인 실패 처리
Completed

로그인 테스트 추가
Running
```

처럼 Task Card 형태로 보여줄 수 있다.

---

# 72. Follow-up Request

사용자가 이전 작업에:

```text
"그런데 잠금 횟수를 3회로 바꿔줘."
```

라고 하면 기존 Work Item을 수정할지 새 Work Item을 만들지는 Server가 판단한다.

IDE는 현재 Conversation/Work Item reference를 전달한다.

---

# 73. Conversation Context

IDE는 모든 채팅을 매 요청마다 전송하지 않는다.

Server가 Conversation을 관리한다.

IDE는 다음만 전달한다.

```text
session_id
conversation_id
active_work_item_id
new message
```

---

# 74. Work Item View

향후 다음 정보를 표시할 수 있다.

```text
Status

Current Stage

Approved Design

Changed Files

Build Result

Test Result
```

Chat 외에 별도 Work Item Panel로 확장 가능하다.

---

# 75. History

완료된 작업 목록:

```text
로그인 실패 정책 구현
09/08 Completed

JWT 설정 검토
09/08 Completed
```

클릭 시 Design/Change/Result를 조회할 수 있다.

---

# 76. Approval History

사용자가 무엇을 승인했는지 확인 가능해야 한다.

예:

```text
Design v2
Approved 14:32

Change Set CHG-104
Approved 14:37
```

감사 추적에도 유용하다.

---

# 77. Source Changed Outside IDE

외부 Editor나 Git 명령으로 파일이 변경될 수도 있다.

IDE Extension은 이를 전부 책임지지 않는다.

Local Agent File Watcher가 공식 변경 감지 주체이다.

---

# 78. IDE Save 이벤트

IDE Save Event를 Local Agent에 전달하면 Index Update를 더 빠르게 Trigger할 수 있다.

하지만 File Watcher가 기본 Source of Truth이다.

---

# 79. Auto Save

VS Code 등의 Auto Save 환경에서도 동일하게 동작해야 한다.

debounce는 Local Agent가 담당한다.

---

# 80. Multi-root Workspace

VS Code Multi-root Workspace 등을 고려한다.

하나의 IDE Window가 여러 프로젝트를 가질 수 있다.

```text
workspace
 ├─ project A
 └─ project B
```

요청 시 active project를 명확히 지정한다.

---

# 81. Project Ambiguity

현재 파일이 없는 상태에서 여러 Project가 열린 경우:

```text
어느 프로젝트에서 작업할까요?

○ backend-service
○ admin-frontend
```

와 같은 선택 UI를 제공한다.

---

# 82. IDE-specific 기능 차이

세 IDE의 기능 차이가 존재하더라도 onCode 핵심 Workflow 의미는 동일해야 한다.

예:

```text
Design Approval

Code Diff Approval

Progress

Question
```

은 모든 IDE에서 제공되어야 한다.

---

# 83. Feature Capability

IDE Extension도 Capability를 Local Agent에 전달할 수 있다.

```json
{
  "capabilities": {
    "native_diff": true,
    "inline_decoration": true,
    "diagnostics": true
  }
}
```

Local Agent/Server가 UI 요청 형식을 조정할 수 있다.

---

# 84. VS Code Extension 구조 예

```text
vscode-extension/

├─ extension
├─ local-agent-client
├─ chat-view
├─ diff
├─ approval
├─ context
└─ commands
```

Webview 사용 영역과 Native API 사용 영역을 구분한다.

---

# 85. IntelliJ Plugin 구조 예

```text
intellij-plugin/

├─ tool-window
├─ editor-context
├─ diff
├─ actions
├─ local-agent-client
└─ notifications
```

---

# 86. Eclipse Plugin 구조 예

```text
eclipse-plugin/

├─ view
├─ editor-context
├─ compare
├─ commands
├─ local-agent-client
└─ notifications
```

---

# 87. 공통 UI Library

IDE 3종이 서로 다른 UI Framework를 사용하므로 UI 코드를 완전히 공유하기는 어렵다.

대신 다음을 공유한다.

```text
Protocol DTO

Message Model

Approval Model

Workflow View Model

Formatting Rules
```

---

# 88. IDE Bridge Protocol DTO

가능하면 별도의 공통 Schema를 정의한다.

예:

```text
ide-message-v1

ide-context-v1

ide-question-v1

ide-approval-v1

ide-diff-v1
```

---

# 89. UI Action Message

예:

```json
{
  "type": "approval.response",

  "approval_id": "APR-100",

  "decision": "REQUEST_CHANGE",

  "reason": "기존 Repository 구조를 사용해주세요."
}
```

---

# 90. IDE Event Message

```json
{
  "type": "ide.event",

  "event": "file.saved",

  "data": {
    "path": "src/main/java/AuthService.java"
  }
}
```

이벤트가 반드시 Server까지 전달되는 것은 아니다.

Local Agent가 필요한 경우에만 처리한다.

---

# 91. Local Agent Event

```json
{
  "type": "agent.event",

  "event": "project.indexing",

  "data": {
    "progress": 64
  }
}
```

IDE에 Project Index 최초 생성 상태를 표시할 수 있다.

---

# 92. 최초 Indexing UX

대형 프로젝트 최초 실행 시:

```text
프로젝트를 분석하고 있습니다.

Files: 1,432 / 2,103
```

등을 표시한다.

분석 중에도 일부 기능은 사용할 수 있지만 Project-aware 요청은 제한할 수 있다.

---

# 93. Indexing 상태

```text
NOT_STARTED

SCANNING

INDEXING

READY

PARTIAL

FAILED
```

IDE Status Bar에 단순 표시할 수 있다.

---

# 94. PROJECT_SUMMARY UI

개발자가 원하면 IDE에서:

```text
onCode: Project Summary 열기
```

명령으로 `.codegen/PROJECT_SUMMARY.md`를 열 수 있다.

---

# 95. .codegen 디렉터리 표시

일반 Project Explorer에서 `.codegen`을 숨길지 여부는 IDE별 정책으로 결정할 수 있다.

기본적으로 실제 Workspace에는 존재하지만 사용자가 원하면 확인 가능해야 한다.

---

# 96. Configuration UI

프로젝트 설정:

```text
Index Ignore

Approval Policy

Build Tool

Summary Level
```

등은 `.codegen/config.yaml`을 수정하도록 할 수 있다.

IDE Settings UI는 이를 편리하게 편집하는 프론트엔드 역할을 한다.

---

# 97. Organization Policy

조직 정책처럼 사용자가 변경할 수 없는 설정은 IDE Settings에서 Read-only로 표시할 수 있다.

예:

```text
Git Push Approval: Required by organization
```

---

# 98. User Preference

사용자 편의 설정은 별도로 허용한다.

예:

```text
Progress detail level

Notification

Auto-open diff

Chat font size
```

보안 정책보다 우선할 수 없다.

---

# 99. Keyboard Shortcut

예:

```text
Ask onCode

Review Selection

Open onCode Chat
```

단축키는 IDE별로 설정 가능하게 한다.

---

# 100. Error Handling

IDE에서는 Error를 세 종류로 구분하여 표시한다.

```text
User Error

Execution Error

System Error
```

예:

```text
요청한 파일을 찾을 수 없습니다.

Build가 실패했습니다.

onCode Server에 연결할 수 없습니다.
```

---

# 101. Protocol Error

IDE가 내부 Protocol Error JSON을 그대로 표시하지 않는다.

예:

```text
PROTOCOL_VERSION_UNSUPPORTED
```

→

```text
onCode Local Agent와 IDE Extension 버전이 호환되지 않습니다.
```

---

# 102. Version Compatibility

IDE Extension과 Local Agent 연결 시 버전을 확인한다.

```text
Extension Protocol: 1.0
Local Agent Protocol: 1.0
```

호환 불가 시 사용자에게 업데이트 필요를 알린다.

---

# 103. 폐쇄망 업데이트

IDE Marketplace 자동 업데이트를 전제하지 않는다.

배포 방식:

```text
VSIX

IntelliJ Plugin Package

Eclipse Update Site / Plugin Package
```

를 사내 배포 체계로 제공한다.

---

# 104. Security

IDE Extension 자체도 최소 권한을 사용한다.

가능한 경우:

```text
필요 Workspace만 접근

임의 Network Access 금지

Server Credential 직접 보관 최소화
```

---

# 105. Prompt Injection UI

Source 또는 문서에 LLM 지시처럼 보이는 문자열이 있어도 IDE는 별도 행동하지 않는다.

이는 데이터일 뿐이다.

---

# 106. Clipboard

onCode가 자동으로 Clipboard에 코드를 복사하거나 읽는 기능은 기본 제공하지 않는다.

사용자 명시 동작이 있을 때만 사용한다.

---

# 107. Telemetry

폐쇄망 환경에서 외부 Telemetry는 사용하지 않는다.

내부 운영 Metrics가 필요하면 Server/Local Agent 내부 수집 체계로 제한한다.

---

# 108. Crash Reporting

외부 Crash 서비스에 자동 전송하지 않는다.

Local Log 또는 내부 서버로만 수집할 수 있다.

---

# 109. IDE Extension Logging

Log 대상:

```text
Connection

Protocol Error

UI Error

Extension Lifecycle
```

Source 전체, Chat 전체, Credential은 기본 Log에서 제외한다.

---

# 110. Accessibility

Approval/Question UI는 Keyboard만으로도 처리할 수 있도록 설계하는 것이 좋다.

---

# 111. Localization

초기에는 한국어/영어 정도를 고려할 수 있다.

Protocol 내부 Enum과 Error Code는 영어 고정값을 사용한다.

UI 문자열만 Localize한다.

---

# 112. Recommended User Flow

```text
Developer
   ↓
IDE Chat
   ↓
"로그인 실패 5회 시 계정을 잠가줘"
   ↓
Progress
   ↓
Question(optional)
   ↓
Design Review
   ↓
Approve
   ↓
Implementation Progress
   ↓
Diff Review
   ↓
Approve
   ↓
Local Apply
   ↓
Build / Test
   ↓
Result
```

---

# 113. Review-only Flow

```text
Developer selects code
       ↓
Review with onCode
       ↓
Analysis
       ↓
Review Findings
       ↓
No Source Modification
```

---

# 114. Find Flow

```text
"권한 체크하는 코드 찾아줘"
       ↓
Project Search
       ↓
Result File List
       ↓
Click
       ↓
Editor Navigation
```

---

# 115. Git Flow

```text
"변경사항 commit 해줘"
       ↓
Git Status
       ↓
Commit Plan
       ↓
Approval
       ↓
Git Commit
       ↓
Result
```

---

# 116. IDE MVP 범위

초기 MVP에서는 다음을 구현한다.

```text
Chat

Current File / Selection Context

Progress

Question

Design Review / Approval

Diff Review / Approval

Build Result

Test Result

Git Approval

File Navigation

Local Agent Connection
```

---

# 117. MVP 후순위

```text
Inline Code Lens

Hunk-level Approval

Work Item Dashboard

Full History Browser

Advanced Diagnostics Integration

Code Action 자동 추천

Multi-root Advanced UI

Local Undo UI

Rich Architecture Visualization
```

---

# 118. Eclipse / VS Code / IntelliJ 우선순위

세 IDE를 모두 최종 지원하더라도 처음부터 동시 구현할 필요는 없다.

권장:

```text
1. VS Code

2. IntelliJ IDEA

3. Eclipse
```

또는 실제 주요 사용자 IDE 점유율에 따라 변경한다.

다만 Protocol과 IDE Adapter Interface는 처음부터 3개 지원을 전제로 설계한다.

---

# 119. 핵심 설계 결정

onCode IDE Extension Architecture v1의 핵심 결정은 다음과 같다.

1. IDE Extension은 Thin Client로 유지한다.
2. IDE는 Server가 아니라 Local Agent에 연결한다.
3. IDE별 차이는 IDE Adapter에 격리한다.
4. Project 분석과 파일 작업은 Local Agent가 담당한다.
5. 사용자 요청 시 현재 파일/선택 범위 등 최소 Context만 전달한다.
6. Source 전체 자동 첨부를 하지 않는다.
7. Progress, Question, Design, Diff, Approval을 구조화된 UI 요소로 제공한다.
8. 설계 승인과 코드 승인 UI를 분리한다.
9. Diff는 Local Agent가 현재 Workspace를 기준으로 생성한다.
10. MVP는 Atomic Change Set Approval을 사용한다.
11. 개발자 반려 사유를 Workflow Constraint로 전달한다.
12. Stale Workspace 발생 시 기존 Diff 적용을 금지한다.
13. Build/Test 결과는 구조화해 표시하고 전체 로그는 별도 제공한다.
14. File/Line 정보는 Editor Navigation과 연결한다.
15. Git/Command 위험 작업에는 명시적인 승인 UI를 제공한다.
16. Local Agent와 IDE Extension은 Local-only 인증된 IPC를 사용한다.
17. Credential과 Server 연결 관리는 Local Agent에 집중한다.
18. IDE 상태 변경 이벤트를 과도하게 Server로 전송하지 않는다.
19. Local Agent File Watcher가 Workspace 변화의 공식 감지 주체이다.
20. IDE Plugin 간 UI 코드는 완전 공유하지 않고 Protocol/View Model을 공유한다.
21. 외부 Telemetry/Crash 전송을 전제하지 않는다.
22. IDE Extension과 Local Agent 버전 호환성을 검사한다.
23. MVP에서는 Chat + Approval + Diff + Result UX에 집중한다.
24. 고급 Inline Agent UX는 후속 버전에서 확장한다.