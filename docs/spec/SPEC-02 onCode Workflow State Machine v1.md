# SPEC-02 onCode Workflow State Machine v1

## 1. 목적

본 명세는 onCode의 사용자 요청이 접수된 시점부터 완료될 때까지의 Workflow 상태와 전이 규칙을 정의한다.

Workflow Engine은 LLM의 판단에 전적으로 의존하지 않고 명시적인 상태 머신을 기반으로 동작한다.

LLM Agent는 다음 단계에서 무엇을 할지 판단할 수 있지만, 실제 상태 전이는 Workflow Engine이 검증하고 수행한다.

---

## 2. 기본 원칙

### 2.1 Workflow와 Agent 분리

Workflow Engine은 작업의 진행 상태를 관리한다.

Agent는 각 상태에서 필요한 판단 및 작업 결과를 생성한다.

```text
Workflow Engine
      ↓
현재 State
      ↓
Agent / Tool 실행
      ↓
Result
      ↓
Transition Rule 평가
      ↓
Next State
```

---

### 2.2 Workflow State는 중앙 서버가 관리

Local Agent나 IDE Extension은 Workflow 상태를 직접 변경하지 않는다.

Local Agent는 Event 또는 Tool Result만 전달한다.

```text
Local Agent
     ↓
Tool Result / Event
     ↓
Server Orchestrator
     ↓
Workflow State Transition
```

---

### 2.3 모든 상태 변경은 Event로 기록

모든 상태 전이는 Audit/Event Store에 기록한다.

예:

```json
{
  "event": "workflow.state.changed",
  "workflow_id": "WF-100",
  "work_item_id": "WI-100",
  "from": "DESIGNING",
  "to": "WAITING_DESIGN_APPROVAL",
  "timestamp": "2026-09-08T15:00:00Z"
}
```

---

# 3. Workflow Scope

사용자 요청 하나는 기본적으로 하나의 `WORK_ITEM`으로 관리한다.

예:

```text
PROJECT PRJ-001

├─ WI-001 로그인 기능 구현
├─ WI-002 AuthService 테스트
└─ WI-003 Git commit 요청
```

하나의 Work Item은 하나 이상의 Task를 포함할 수 있다.

```text
WORK_ITEM
   ↓
TASK
   ├─ GUIDE_SEARCH
   ├─ DESIGN
   ├─ IMPLEMENTATION
   ├─ REVIEW
   └─ TEST
```

---

# 4. Workflow Type

요청 종류에 따라 Workflow 유형을 구분한다.

```text
GENERAL
ANALYSIS
REVIEW
IMPLEMENTATION
FIX
TEST
BUILD
GIT
SECURITY
DEPENDENCY
```

---

# 5. 공통 상태

모든 Workflow는 다음 공통 상태를 사용할 수 있다.

```text
CREATED
RECEIVED
CLASSIFYING
RUNNING
WAITING_USER_INPUT
WAITING_APPROVAL
COMPLETED
FAILED
CANCELLED
```

---

# 6. Implementation Workflow 상세 상태

구현 및 수정 요청에 사용하는 기본 Workflow이다.

```text
RECEIVED
   ↓
CLASSIFYING
   ↓
DISCOVERING_CONTEXT
   ↓
ANALYZING_REQUIREMENT
   ↓
RETRIEVING_GUIDE
   ↓
DESIGNING
   ↓
WAITING_DESIGN_APPROVAL
   ↓
ANALYZING_DEPENDENCY
   ↓
IMPLEMENTING
   ↓
REVIEWING
   ↓
SECURITY_REVIEWING
   ↓
PREPARING_CHANGE
   ↓
WAITING_CODE_APPROVAL
   ↓
APPLYING_CHANGE
   ↓
VALIDATING_SYNTAX
   ↓
BUILDING
   ↓
TESTING
   ↓
FINAL_REVIEW
   ↓
COMPLETED
```

---

# 7. RECEIVED

사용자의 요청이 Server에 정상 접수된 상태이다.

입력:

```text
user request
session
project
IDE context
selection context
```

처리:

```text
Work Item 생성
Workflow 생성
Audit 기록
```

다음 상태:

```text
CLASSIFYING
```

---

# 8. CLASSIFYING

사용자 요청의 intent 및 Workflow Type을 결정한다.

예:

```json
{
  "intent": "MODIFY",
  "workflow_type": "IMPLEMENTATION",
  "requires_project_context": true,
  "requires_design": true,
  "requires_approval": true,
  "requires_build": true,
  "requires_test": true
}
```

가능한 전이:

```text
GENERAL
→ ANSWERING

IMPLEMENTATION
→ DISCOVERING_CONTEXT

TEST
→ PREPARING_TEST

BUILD
→ PREPARING_BUILD

GIT
→ PREPARING_GIT

REVIEW
→ DISCOVERING_CONTEXT
```

---

# 9. DISCOVERING_CONTEXT

Project Catalog를 이용하여 필요한 프로젝트 정보를 탐색한다.

기본 Tool:

```text
project.get_profile
project.search
project.get_symbols
project.get_dependencies
```

필요한 파일을 확인한 후:

```text
workspace.read_file
workspace.read_files
```

를 사용한다.

---

## 9.1 프로젝트 정보 부족

Project Index가 없거나 오래된 경우:

```text
PROJECT_INDEX_REQUIRED
```

상태로 전환한다.

Local Agent에:

```text
project.get_index
```

또는 재생성을 요청한다.

---

## 9.2 Stale Project Index

서버 revision:

```text
103
```

Local Agent revision:

```text
110
```

관련 파일이 변경되었으면:

```text
STALE_WORKSPACE
```

처리 후 Context Discovery를 다시 수행한다.

---

# 10. ANALYZING_REQUIREMENT

Requirement Agent가 사용자 요청을 구조화한다.

결과 예:

```json
{
  "goal": "로그인 실패 5회 시 계정 잠금",

  "functional_requirements": [
    "실패 횟수를 관리한다.",
    "5회 이상 실패하면 잠금 처리한다."
  ],

  "constraints": [
    "현재 인증 구조 유지",
    "기존 개발 가이드 준수"
  ],

  "unknowns": [
    "잠금 해제 정책"
  ]
}
```

---

# 11. 요구사항 모호성 처리

`unknowns`가 구현 결과에 영향을 주는 경우 사용자에게 질문한다.

상태:

```text
WAITING_USER_INPUT
```

사용 Tool:

```text
user.ask
```

예:

```text
계정 잠금 해제 정책을 선택해 주세요.

A. 관리자 해제
B. 30분 후 자동
C. 기존 프로젝트 정책 사용
```

사용자 답변은 Context Storage에 `USER_DECISION`으로 저장한다.

응답 후:

```text
ANALYZING_REQUIREMENT
```

또는:

```text
DESIGNING
```

으로 복귀한다.

---

# 12. RETRIEVING_GUIDE

구현에 필요한 개발 가이드를 조회한다.

Tool:

```text
guide.search
guide.read
```

검색 대상:

```text
Language Guide
Framework Guide
Security Guide
Project Guide
Organization Guide
```

---

# 13. Guide 우선순위

충돌 시 다음 순서를 적용한다.

```text
PROJECT_RULE

    >

ORGANIZATION_RULE

    >

FRAMEWORK_RULE

    >

LANGUAGE_RULE

    >

GENERAL_RULE
```

---

# 14. Guide 미발견

필요한 개발 가이드가 없으면 다음 중 정책에 따라 처리한다.

```text
GUIDE_NOT_FOUND
```

선택 정책:

```text
ALLOW_WITH_WARNING
ASK_USER
BLOCK_IMPLEMENTATION
```

프로젝트별 설정으로 제어할 수 있다.

---

# 15. DESIGNING

Senior Developer / Design Agent가 구현 설계를 생성한다.

Design에 최소 포함할 정보:

```text
목표
변경 범위
대상 파일
추가 파일
수정 클래스
수정 메서드
데이터 흐름
Exception 처리
Dependency
Security 고려
Test 전략
Guide Reference
```

---

# 16. Design Version

모든 Design은 version을 가진다.

```text
DESIGN v1
DESIGN v2
DESIGN v3
```

Context Storage:

```text
ctx://work-items/WI-100/design/1
ctx://work-items/WI-100/design/2
ctx://work-items/WI-100/design/latest
```

---

# 17. WAITING_DESIGN_APPROVAL

개발자에게 설계를 전달한다.

Tool:

```text
user.request_approval
```

Approval Type:

```text
DESIGN
```

지원 Action:

```text
APPROVE
REQUEST_CHANGE
REJECT
CANCEL
```

---

# 18. Design Approve

```text
WAITING_DESIGN_APPROVAL
        ↓ APPROVE
ANALYZING_DEPENDENCY
```

승인 결과를 `USER_DECISION`과 `APPROVAL`에 기록한다.

---

# 19. Design Change Request

예:

```text
"DB 스키마 변경 없이 구현해줘."
```

전이:

```text
WAITING_DESIGN_APPROVAL
        ↓ REQUEST_CHANGE
ANALYZING_REQUIREMENT
        ↓
DESIGNING
```

사용자의 변경 요청은 새로운 Constraint로 저장한다.

---

# 20. Design Reject

완전 반려:

```text
WAITING_DESIGN_APPROVAL
       ↓ REJECT
REJECTED
```

다만 반려 사유가 재작업 요청에 가까운 경우:

```text
ANALYZING_REQUIREMENT
```

로 되돌릴 수 있다.

---

# 21. ANALYZING_DEPENDENCY

구현에 필요한 라이브러리를 검토한다.

우선순위:

```text
1. 기존 프로젝트 Dependency
2. Nexus에 이미 승인된 Dependency
3. 대체 Dependency
4. 다른 구현 방식
5. 자체 구현 검토
```

---

# 22. Dependency Workflow

```text
현재 Dependency 확인
       ↓
기존 Library 사용 가능?
     /       \
   YES       NO
    │         │
    │      Nexus Search
    │         │
    │      Candidate
    │         │
    └──────► Security Check
```

Tool:

```text
dependency.search
dependency.resolve
security.check_dependency
```

---

# 23. Vulnerable Dependency

취약점 발견 시 기본 정책:

```text
안전한 Version 검색
      ↓
대체 Library 검색
      ↓
Library 없는 설계 검토
      ↓
자체 구현 가능성 검토
```

취약 라이브러리를 임의로 사용하는 것은 허용하지 않는다.

상태:

```text
DEPENDENCY_REJECTED
```

필요하면:

```text
DESIGNING
```

으로 돌아간다.

---

# 24. IMPLEMENTING

Implementation Agent가 코드를 생성한다.

Agent는 Workspace를 직접 수정하지 않는다.

출력은 반드시:

```text
Proposed Change Set
```

이어야 한다.

예:

```json
{
  "change_set_id": "CHG-100",

  "changes": [
    {
      "path": "AuthService.java",
      "operation": "MODIFY",
      "base_hash": "sha256:...",
      "content": "..."
    }
  ]
}
```

---

# 25. REVIEWING

Implementation Agent와 다른 Review Agent가 코드를 검토한다.

검토 기준:

```text
Requirement
Design
Guide
Architecture
Naming
Exception Handling
Duplication
Backward Compatibility
Testability
```

---

# 26. Review Failed

```text
REVIEWING
    ↓ FAILED
IMPLEMENTING
```

필요한 경우:

```text
DESIGNING
```

까지 되돌릴 수 있다.

예:

```text
단순 구현 오류
→ IMPLEMENTING

설계 자체 문제
→ DESIGNING
```

---

# 27. SECURITY_REVIEWING

Security Agent가 코드 및 Dependency를 검토한다.

검토 항목:

```text
Dependency Vulnerability
Injection
Authentication
Authorization
Credential Exposure
Path Traversal
Unsafe Command
Deserialization
Cryptography
Secret Leakage
```

---

# 28. Security Failed

단순 코드 보안 문제:

```text
SECURITY_REVIEWING
        ↓
IMPLEMENTING
```

구조적 문제:

```text
SECURITY_REVIEWING
        ↓
DESIGNING
```

Dependency 문제:

```text
SECURITY_REVIEWING
        ↓
ANALYZING_DEPENDENCY
```

---

# 29. PREPARING_CHANGE

검증이 완료된 Proposed Change를 Local Agent에 전달한다.

Tool:

```text
workspace.propose_changes
```

Local Agent가 현재 Workspace와 비교해서 Diff를 생성한다.

---

# 30. WAITING_CODE_APPROVAL

IDE Extension에 Diff를 표시한다.

사용자는 다음 작업을 수행할 수 있다.

```text
APPROVE_ALL
APPROVE_FILE
REJECT_FILE
REQUEST_CHANGE
REJECT_ALL
```

---

# 31. Code Approve

```text
WAITING_CODE_APPROVAL
       ↓ APPROVE
APPLYING_CHANGE
```

Approval ID가 생성된다.

---

# 32. 부분 승인

여러 파일 중 일부만 승인 가능하도록 설계할 수 있다.

예:

```text
AuthService.java          APPROVED
UserRepository.java       APPROVED
UserController.java       REJECTED
```

이 경우 기본적으로:

```text
PARTIAL_APPROVAL
```

로 처리한다.

부분 적용이 논리적으로 안전한 경우에만 허용한다.

기본 정책은:

```text
ATOMIC_CHANGE_SET
```

을 권장한다.

즉 하나의 Change Set 전체 승인 또는 전체 반려가 기본이다.

---

# 33. Code Reject

사용자 반려 사유:

```text
"새 클래스를 만들지 말고 기존 AuthService에서 처리해줘."
```

전이:

```text
WAITING_CODE_APPROVAL
        ↓
ANALYZING_REQUIREMENT
        ↓
DESIGNING
```

코드만 다시 생성하는 것이 아니라 설계부터 재검토한다.

---

# 34. APPLYING_CHANGE

Tool:

```text
workspace.apply_changes
```

실행 전 반드시 확인:

```text
base_hash == current_hash
```

---

# 35. Workspace Changed

개발자가 승인 과정 중 파일을 수정한 경우:

```text
WORKSPACE_FILE_CHANGED
```

상태:

```text
STALE_WORKSPACE
```

전이:

```text
STALE_WORKSPACE
      ↓
DISCOVERING_CONTEXT
```

기존 Change Set은 폐기한다.

---

# 36. VALIDATING_SYNTAX

파일 적용 직후 가장 빠른 검증을 수행한다.

언어별 예:

```text
Java
→ syntax / compiler frontend

Python
→ compile / syntax check

JavaScript / TypeScript
→ parser / tsc

Vue
→ SFC parser

React
→ JS/TS syntax
```

문법 오류가 발견되면:

```text
IMPLEMENTING
```

으로 돌아간다.

---

# 37. BUILDING

Tool:

```text
build.run
```

빌드 결과는 구조화해서 반환한다.

성공:

```text
BUILDING
   ↓
TESTING
```

실패:

```text
BUILD_FAILED
```

---

# 38. BUILD_FAILED

Local Agent가 다음 형태로 오류를 전달한다.

```json
{
  "file": "AuthService.java",
  "line": 84,
  "type": "COMPILATION_ERROR",
  "message": "cannot find symbol"
}
```

전체 로그는 필요할 경우 별도로 조회한다.

---

# 39. Build Failure 분류

Senior Developer Agent가 원인을 분류한다.

```text
CODE_ERROR
DEPENDENCY_ERROR
ENVIRONMENT_ERROR
CONFIGURATION_ERROR
INFRASTRUCTURE_ERROR
```

---

# 40. Build Failure Transition

코드 오류:

```text
BUILD_FAILED
    ↓
IMPLEMENTING
```

Dependency 오류:

```text
BUILD_FAILED
    ↓
ANALYZING_DEPENDENCY
```

환경 문제:

```text
BUILD_FAILED
    ↓
WAITING_USER_INPUT
```

예:

```text
JDK 17이 필요하지만 로컬에는 JDK 11만 설치되어 있습니다.
```

---

# 41. TESTING

Tool:

```text
test.run
```

기본적으로 변경 코드와 관련된 테스트부터 수행한다.

```text
RELATED
```

필요할 경우:

```text
MODULE
PROJECT
```

로 확대한다.

---

# 42. TEST_FAILED

실패 유형:

```text
IMPLEMENTATION_ERROR
TEST_ERROR
EXISTING_FAILURE
ENVIRONMENT_ERROR
FLAKY_TEST
```

---

# 43. Test Failure Transition

구현 오류:

```text
TEST_FAILED
    ↓
IMPLEMENTING
```

설계 오류:

```text
TEST_FAILED
    ↓
DESIGNING
```

테스트 코드 문제:

```text
TEST_FAILED
    ↓
IMPLEMENTING
```

기존 실패:

```text
TEST_FAILED
    ↓
WAITING_USER_INPUT
```

또는 정책에 따라 warning 후 완료할 수 있다.

---

# 44. FINAL_REVIEW

모든 변경과 실행 결과를 최종 점검한다.

확인 항목:

```text
Requirement 충족

Approved Design 일치

Guide Compliance

Security Result

Build Result

Test Result

Workspace Revision

승인된 파일 외 변경 여부
```

---

# 45. COMPLETED

정상 종료 상태.

최종 결과에는 최소 다음을 포함한다.

```text
변경 파일
주요 변경 내용
Build 결과
Test 결과
Guide Reference
Dependency 변경
Security Findings
사용자 승인 정보
```

---

# 46. General QA Workflow

일반적인 질문에는 Full Development Workflow를 사용하지 않는다.

```text
RECEIVED
   ↓
CLASSIFYING
   ↓
ANSWERING
   ↓
COMPLETED
```

예:

```text
"Spring Transaction이 뭐야?"
```

---

# 47. Project Q&A Workflow

프로젝트에 대한 질문:

```text
RECEIVED
   ↓
CLASSIFYING
   ↓
DISCOVERING_CONTEXT
   ↓
ANALYZING
   ↓
ANSWERING
   ↓
COMPLETED
```

예:

```text
"우리 프로젝트 로그인 처리가 어떻게 되어 있어?"
```

---

# 48. Code Review Workflow

```text
RECEIVED
   ↓
CLASSIFYING
   ↓
DISCOVERING_CONTEXT
   ↓
RETRIEVING_GUIDE
   ↓
REVIEWING
   ↓
SECURITY_REVIEWING
   ↓
ANSWERING
   ↓
COMPLETED
```

파일 변경은 발생하지 않는다.

---

# 49. Build-only Workflow

사용자 요청:

```text
"프로젝트 빌드해줘."
```

Workflow:

```text
RECEIVED
   ↓
CLASSIFYING
   ↓
PREPARING_BUILD
   ↓
WAITING_EXECUTION_APPROVAL
   ↓
BUILDING
   ↓
ANALYZING_RESULT
   ↓
COMPLETED
```

---

# 50. Test-only Workflow

```text
RECEIVED
   ↓
CLASSIFYING
   ↓
PREPARING_TEST
   ↓
WAITING_EXECUTION_APPROVAL
   ↓
TESTING
   ↓
ANALYZING_RESULT
   ↓
COMPLETED
```

---

# 51. Git Workflow

예:

```text
"feature/login 브랜치 만들어줘."
```

Workflow:

```text
RECEIVED
   ↓
CLASSIFYING
   ↓
PREPARING_GIT
   ↓
POLICY_CHECK
   ↓
WAITING_APPROVAL
   ↓
EXECUTING_GIT
   ↓
VERIFYING_GIT_RESULT
   ↓
COMPLETED
```

---

# 52. Git 위험도

Read 작업:

```text
git.status
git.diff
git.log
```

→ 승인 없이 실행 가능

변경 작업:

```text
git.checkout
git.add
```

→ 정책에 따라 승인

High Risk:

```text
git.commit
git.merge
git.rebase
git.push
```

→ 명시적 승인

Destructive:

```text
git.reset --hard
git.clean -fd
```

→ 강한 승인 또는 정책 차단

---

# 53. WAITING_USER_INPUT

사용자의 추가 정보가 필요한 상태이다.

이 상태에서는 Workflow가 진행되지 않는다.

질문 예:

```text
"DB 스키마 변경이 가능한가요?"

"기존 인증 정책을 유지할까요?"
```

사용자 응답이 오면 이전 state 또는 명시된 resume state로 복귀한다.

---

# 54. WAITING_APPROVAL

Approval을 기다리는 공통 상태이다.

Approval Type:

```text
DESIGN
CODE_CHANGE
COMMAND
DEPENDENCY_INSTALL
GIT_COMMIT
GIT_PUSH
DESTRUCTIVE_OPERATION
```

---

# 55. CANCELLED

다음 상황에서 전환할 수 있다.

```text
사용자 취소

Session 종료

Workflow 관리자 취소

시스템 정책 취소
```

실행 중 Tool이 있다면:

```text
CANCEL
```

메시지를 전달한다.

---

# 56. FAILED

복구 불가능한 오류가 발생한 최종 상태이다.

예:

```text
프로토콜 오류

인증 오류

반복 실패 횟수 초과

필수 Tool 없음

Local Agent 연결 불가

Context Storage 장애
```

---

# 57. Retry 정책

일시 오류는 자동 Retry 가능하다.

```text
NETWORK_ERROR
TEMPORARY_UNAVAILABLE
TIMEOUT
```

예:

```text
max_retry = 3
```

권장 backoff:

```text
1초
3초
10초
```

---

# 58. Retry 금지 오류

```text
PERMISSION_DENIED

APPROVAL_REJECTED

SECURITY_POLICY_VIOLATION

INVALID_ARGUMENT

DEPENDENCY_VULNERABLE
```

은 자동 retry하지 않는다.

---

# 59. Infinite Loop 방지

Agent 수정 Loop에는 제한을 둔다.

예:

```text
IMPLEMENTATION_RETRY_LIMIT = 5

BUILD_FIX_LIMIT = 3

TEST_FIX_LIMIT = 3

DESIGN_REVISION_LIMIT = configurable
```

초과 시:

```text
WAITING_USER_INPUT
```

또는:

```text
FAILED
```

로 전환한다.

---

# 60. Workflow Resume

서버 재시작 또는 Local Agent 연결 단절 후 Workflow를 복구할 수 있어야 한다.

Workflow DB에는 다음 정보를 저장한다.

```text
current_state

previous_state

resume_state

workflow_revision

pending_tool_call

pending_approval

context_refs
```

---

# 61. Local Agent Disconnect

Local Agent와 연결이 끊기면:

```text
LOCAL_AGENT_UNAVAILABLE
```

상태로 두거나 Workflow를 일시 중단한다.

재연결 후:

```text
system.get_capabilities
project revision 확인
workspace 상태 확인
```

후 Resume한다.

---

# 62. Approval 만료

승인 요청에는 선택적으로 유효기간을 둘 수 있다.

```text
WAITING_CODE_APPROVAL
        ↓ timeout
APPROVAL_EXPIRED
```

파일 상태가 바뀌었을 가능성이 있으므로 자동 승인 복구는 금지한다.

---

# 63. Concurrency

동일 프로젝트에서 여러 Workflow가 동시에 실행될 수 있다.

예:

```text
WF-100
AuthService 수정

WF-101
UserService 수정
```

파일 범위가 겹치지 않으면 병렬 진행 가능하다.

---

# 64. 파일 충돌

두 Workflow가 동일 파일을 수정하려는 경우:

```text
FILE_CONFLICT
```

을 발생시킨다.

기본 정책:

```text
첫 번째 Workflow가 Write Lock 획득

다른 Workflow는 WAITING_RESOURCE 상태
```

또는 적용 단계에서 hash mismatch로 탐지할 수 있다.

---

# 65. Workspace Lock

프로젝트 전체 Lock보다는 파일 단위 또는 Change Set 단위 Soft Lock을 권장한다.

```text
AuthService.java
Owner: WF-100

UserService.java
Owner: WF-101
```

개발자가 직접 IDE에서 파일을 수정하는 것은 차단하지 않는다.

대신 hash/revision으로 충돌을 탐지한다.

---

# 66. 상태 전이 데이터 모델

각 Transition은 다음 정보를 기록한다.

```json
{
  "workflow_id": "WF-100",

  "from_state": "BUILDING",

  "to_state": "BUILD_FAILED",

  "trigger": "TOOL_RESULT",

  "trigger_id": "CALL-300",

  "reason": "COMPILATION_ERROR",

  "timestamp": "2026-09-08T15:42:00Z"
}
```

---

# 67. Workflow Progress

Workflow 상태는 IDE에 Progress Event로 전달한다.

예:

```text
✓ 요청 분석
✓ 프로젝트 탐색
✓ 개발 가이드 확인
✓ 설계
✓ 설계 승인
● 코드 구현
○ 코드 검토
○ 보안 검토
○ 코드 승인
○ 빌드
○ 테스트
```

---

# 68. 상태와 사용자 메시지 분리

내부 상태 이름을 그대로 사용자에게 보여줄 필요는 없다.

예:

```text
ANALYZING_DEPENDENCY
```

IDE 표시:

```text
"필요한 라이브러리와 버전을 검토하고 있습니다."
```

---

# 69. Workflow State Group

UI와 모니터링에서는 세부 상태를 그룹화할 수 있다.

## Planning

```text
CLASSIFYING
DISCOVERING_CONTEXT
ANALYZING_REQUIREMENT
RETRIEVING_GUIDE
DESIGNING
```

## Implementation

```text
ANALYZING_DEPENDENCY
IMPLEMENTING
REVIEWING
SECURITY_REVIEWING
```

## Approval

```text
WAITING_USER_INPUT
WAITING_DESIGN_APPROVAL
WAITING_CODE_APPROVAL
WAITING_EXECUTION_APPROVAL
```

## Execution

```text
APPLYING_CHANGE
VALIDATING_SYNTAX
BUILDING
TESTING
```

## Final

```text
FINAL_REVIEW
COMPLETED
FAILED
CANCELLED
```

---

# 70. Workflow Engine과 Context Storage

Workflow Engine은 실제 대용량 Context를 보유하지 않는다.

```text
Workflow
 ├─ current state
 ├─ transition history
 └─ Context Reference
```

실제 내용:

```text
Requirement
Design
User Decision
Build Result
Test Result
```

은 Context Storage에 보관한다.

---

# 71. Workflow Entity 예

```json
{
  "workflow_id": "WF-100",

  "work_item_id": "WI-100",

  "workflow_type": "IMPLEMENTATION",

  "current_state": "IMPLEMENTING",

  "previous_state": "ANALYZING_DEPENDENCY",

  "revision": 17,

  "status": "RUNNING",

  "context_refs": {
    "requirement": "ctx://work-items/WI-100/requirement/latest",
    "design": "ctx://work-items/WI-100/design/2"
  },

  "created_at": "...",
  "updated_at": "..."
}
```

---

# 72. Workflow Transition Guard

State 전환 전에 Guard를 확인한다.

예:

```text
WAITING_DESIGN_APPROVAL
        ↓
ANALYZING_DEPENDENCY
```

Guard:

```text
design.approval.status == APPROVED
```

코드 적용:

```text
WAITING_CODE_APPROVAL
        ↓
APPLYING_CHANGE
```

Guard:

```text
approval.status == APPROVED
AND
change_set.base_hash == workspace.current_hash
```

---

# 73. 주요 Transition Guard

### Design 승인

```text
approved_design_exists == true
```

### Implementation

```text
requirement_confirmed == true
AND
design_approved == true
```

### Code Apply

```text
code_approved == true
AND
workspace_not_stale == true
```

### Build

```text
change_applied == true
AND
syntax_valid == true
```

### Complete

```text
required_build_success == true
AND
required_test_success == true
AND
final_review_passed == true
```

---

# 74. Implementation Workflow 전체 요약

```text
RECEIVED
   │
   ▼
CLASSIFYING
   │
   ▼
DISCOVERING_CONTEXT
   │
   ▼
ANALYZING_REQUIREMENT
   │
   ├── ambiguity ──► WAITING_USER_INPUT
   │                     │
   │                     └────────────┐
   ▼                                  │
RETRIEVING_GUIDE ◄────────────────────┘
   │
   ▼
DESIGNING
   │
   ▼
WAITING_DESIGN_APPROVAL
   │
   ├── change request ──► ANALYZING_REQUIREMENT
   │
   ▼
ANALYZING_DEPENDENCY
   │
   ▼
IMPLEMENTING ◄────────────────────────────┐
   │                                      │
   ▼                                      │
REVIEWING ───────── failed ───────────────┤
   │                                      │
   ▼                                      │
SECURITY_REVIEWING ─ failed ──────────────┤
   │                                      │
   ▼                                      │
PREPARING_CHANGE                          │
   │                                      │
   ▼                                      │
WAITING_CODE_APPROVAL                     │
   │                                      │
   ├── reject ──► ANALYZING_REQUIREMENT   │
   │                                      │
   ▼                                      │
APPLYING_CHANGE                           │
   │                                      │
   ├── stale ───► DISCOVERING_CONTEXT     │
   │                                      │
   ▼                                      │
VALIDATING_SYNTAX ─ failed ───────────────┤
   │                                      │
   ▼                                      │
BUILDING ───────── code error ────────────┤
   │                                      │
   ▼                                      │
TESTING ────────── implementation error ──┘
   │
   ▼
FINAL_REVIEW
   │
   ▼
COMPLETED
```

---

# 75. MVP 구현 우선순위

초기 MVP에서는 다음 상태를 먼저 구현하는 것을 권장한다.

```text
RECEIVED

CLASSIFYING

DISCOVERING_CONTEXT

ANALYZING_REQUIREMENT

WAITING_USER_INPUT

RETRIEVING_GUIDE

DESIGNING

WAITING_DESIGN_APPROVAL

IMPLEMENTING

REVIEWING

WAITING_CODE_APPROVAL

APPLYING_CHANGE

BUILDING

TESTING

COMPLETED

FAILED

CANCELLED
```

다음은 2차 단계로 확장한다.

```text
SECURITY_REVIEWING

ANALYZING_DEPENDENCY

STALE_WORKSPACE

PARTIAL_APPROVAL

WAITING_RESOURCE

Workflow Resume

Concurrent Workflow Control
```

---

# 76. 핵심 설계 결정

onCode Workflow State Machine v1의 핵심 결정은 다음과 같다.

1. Workflow Engine과 LLM Agent를 분리한다.
2. 상태 전이는 Server Orchestrator만 수행한다.
3. 설계 승인과 코드 승인 단계를 분리한다.
4. 코드 반려 시 단순 재생성이 아닌 요구사항/설계 단계로 회귀한다.
5. File Hash와 Project Revision으로 Stale Workspace를 감지한다.
6. Build/Test 오류는 구조화해서 Senior Developer에게 전달한다.
7. Agent 수정 Loop에 최대 횟수를 둔다.
8. 모든 상태와 Transition을 Audit Event로 기록한다.
9. Context 데이터 자체가 아닌 Context Reference를 Workflow가 관리한다.
10. 상태 전환 전 Guard 조건을 검증한다.
11. 일반 질문과 개발 Workflow를 분리한다.
12. Local Tool 실행 결과로 Workflow 상태가 변경된다.
13. 사용자 질문과 승인을 정식 Workflow State로 처리한다.
14. 장시간 Workflow는 Resume 가능하도록 상태를 영속화한다.
15. 동시 작업에서는 File Conflict와 Workspace Revision을 관리한다.