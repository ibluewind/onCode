# onCode Tool Protocol v1 Specification

## 1. 목적

onCode Tool Protocol은 폐쇄망 환경에서 동작하는 onCode의 다음 구성요소 간 상호작용을 표준화하기 위한 JSON 기반 프로토콜이다.

- 중앙 Orchestrator ↔ Server Agent
- Server Agent ↔ Server Tool
- Orchestrator ↔ Local Agent
- Server Agent ↔ Local Agent Tool
- Local Agent ↔ IDE Extension
- Workflow Engine ↔ Human Approval Interface

프로토콜의 주요 목적은 다음과 같다.

1. 중앙 서버와 Local Agent 간 실행 인터페이스 표준화
2. Agent 구현체 간 결합도 최소화
3. 모든 작업의 추적성과 감사 가능성 확보
4. Human-in-the-Loop 승인 처리 지원
5. Context Storage 기반 Agent 협업 지원
6. Tool 실행 결과의 구조화
7. 취소, 오류, timeout 및 재시도 처리 표준화
8. 향후 MCP 등 외부 Tool Protocol과의 Adapter 구현 용이성 확보

---

# 2. 기본 설계 원칙

## 2.1 Transport Independent

Tool Protocol 자체는 특정 통신 기술에 종속되지 않는다.

사용 가능한 Transport 예:

- HTTPS
- WebSocket
- gRPC
- Message Queue

예를 들어 Server ↔ Local Agent는 WebSocket 또는 gRPC를 사용할 수 있지만 Tool Payload 자체는 동일한 JSON 구조를 유지한다.

---

## 2.2 JSON 기반

모든 Tool Request, Response 및 Event는 JSON 형식을 기본으로 한다.

UTF-8을 사용한다.

---

## 2.3 Tool 중심 구조

Agent가 다른 Agent의 내부 구현을 직접 호출하지 않는다.

다음과 같이 호출한다.

```text
Senior Developer Agent
        ↓
project.search
        ↓
Project Catalog Tool
```

또는:

```text
Senior Developer Agent
        ↓
workspace.read_files
        ↓
Local Agent
```

Agent는 Tool 제공자의 위치가 Local인지 Server인지 알 필요가 없어야 한다.

---

## 2.4 Agent 간 Context 직접 전달 금지

Agent A가 Agent B에게 전체 Prompt 또는 전체 Context를 직접 전달하지 않는다.

```text
Agent A
   ↓
Context Storage
   ↑
Agent B
```

Tool 호출에서는 필요한 Context의 식별자만 전달할 수 있다.

예:

```json
{
  "context_refs": [
    "ctx://work-items/WI-100/design/latest",
    "ctx://work-items/WI-100/user-decisions"
  ]
}
```

---

## 2.5 최소 데이터 전달

서버는 프로젝트 전체 소스를 요청하지 않는다.

우선 Project Index를 검색한다.

```text
project.search
      ↓
필요 파일 식별
      ↓
workspace.read_files
```

방식을 기본 원칙으로 한다.

---

## 2.6 모든 변경 작업은 추적 가능해야 한다

파일 수정, 명령 실행, Git 작업 등 상태를 변경하는 작업은 최소한 다음 정보를 기록해야 한다.

- 사용자
- Session
- Project
- Workflow
- Work Item
- Tool
- Arguments
- 승인 여부
- 실행 시간
- 실행 결과
- 오류
- 대상 파일
- 변경 hash

---

# 3. Protocol Version

초기 버전:

```text
oncode-tool/1.0
```

모든 메시지는 다음 필드를 포함한다.

```json
{
  "protocol": "oncode-tool/1.0"
}
```

Minor Version은 하위 호환 변경에 사용한다.

```text
1.0
1.1
1.2
```

호환되지 않는 변경은 Major Version을 변경한다.

```text
2.0
```

---

# 4. Message Type

Protocol v1은 다음 메시지를 정의한다.

```text
REQUEST

RESPONSE

EVENT

CANCEL

ACK
```

---

# 5. 공통 Message Envelope

기본 Envelope는 다음 구조를 사용한다.

```json
{
  "protocol": "oncode-tool/1.0",
  "message_type": "REQUEST",

  "message_id": "MSG-01J...",
  "timestamp": "2026-09-08T14:30:00.000Z",

  "session_id": "SES-001",
  "project_id": "PRJ-001",
  "workflow_id": "WF-001",
  "work_item_id": "WI-001",
  "task_id": "TASK-001",

  "actor": {
    "type": "AGENT",
    "id": "senior-developer-agent"
  },

  "target": {
    "type": "TOOL",
    "id": "workspace.read_files"
  },

  "payload": {}
}
```

---

# 6. 식별자

권장 식별자는 UUIDv7 또는 이에 준하는 시간 순서 기반 ID이다.

논리적 prefix를 추가할 수 있다.

예:

```text
MSG-...
SES-...
PRJ-...
WF-...
WI-...
TASK-...
CALL-...
DEC-...
APR-...
```

---

# 7. Actor

Tool 호출 주체를 나타낸다.

```json
{
  "actor": {
    "type": "AGENT",
    "id": "senior-developer-agent"
  }
}
```

지원 type:

```text
USER
AGENT
ORCHESTRATOR
LOCAL_AGENT
IDE
SYSTEM
```

---

# 8. Tool Request

Tool 호출의 기본 구조는 다음과 같다.

```json
{
  "protocol": "oncode-tool/1.0",
  "message_type": "REQUEST",

  "message_id": "MSG-1001",
  "timestamp": "2026-09-08T14:30:00Z",

  "session_id": "SES-100",
  "project_id": "PRJ-200",
  "workflow_id": "WF-300",
  "work_item_id": "WI-400",
  "task_id": "TASK-500",

  "actor": {
    "type": "AGENT",
    "id": "senior-developer-agent"
  },

  "target": {
    "type": "TOOL",
    "id": "workspace.read_files"
  },

  "payload": {
    "call_id": "CALL-600",

    "tool": "workspace.read_files",

    "arguments": {
      "paths": [
        "src/main/java/com/example/AuthService.java"
      ]
    },

    "options": {
      "timeout_ms": 30000
    }
  }
}
```

---

# 9. Tool Response

성공 응답:

```json
{
  "protocol": "oncode-tool/1.0",
  "message_type": "RESPONSE",

  "message_id": "MSG-1002",
  "timestamp": "2026-09-08T14:30:01Z",

  "session_id": "SES-100",
  "project_id": "PRJ-200",
  "workflow_id": "WF-300",
  "work_item_id": "WI-400",
  "task_id": "TASK-500",

  "payload": {
    "call_id": "CALL-600",

    "status": "SUCCESS",

    "result": {
      "files": [
        {
          "path": "src/main/java/com/example/AuthService.java",
          "hash": "sha256:abc123",
          "content": "..."
        }
      ]
    },

    "metrics": {
      "duration_ms": 87
    }
  }
}
```

---

# 10. Tool Status

공통 status는 다음과 같이 정의한다.

```text
SUCCESS

FAILED

PARTIAL

ACCEPTED

RUNNING

CANCELLED

REJECTED

TIMEOUT
```

`ACCEPTED`는 실행 요청은 접수되었으나 작업이 아직 완료되지 않은 경우 사용한다.

---

# 11. 비동기 작업

Build, Test 등 오래 걸리는 작업은 동기 Response를 기다리지 않는 방식을 지원한다.

Request:

```json
{
  "tool": "build.run",
  "arguments": {
    "build_tool": "maven"
  }
}
```

응답:

```json
{
  "status": "ACCEPTED",
  "result": {
    "execution_id": "EXEC-1001"
  }
}
```

이후:

```text
tool.started

tool.progress

tool.completed
```

Event를 전달한다.

---

# 12. Event Message

예:

```json
{
  "protocol": "oncode-tool/1.0",
  "message_type": "EVENT",

  "message_id": "MSG-2001",
  "timestamp": "2026-09-08T14:31:00Z",

  "session_id": "SES-100",
  "project_id": "PRJ-200",
  "workflow_id": "WF-300",
  "work_item_id": "WI-400",

  "payload": {
    "event": "tool.progress",

    "execution_id": "EXEC-1001",

    "data": {
      "progress": 60,
      "message": "Compiling Java sources"
    }
  }
}
```

---

# 13. 주요 Event 종류

## Workflow

```text
workflow.started
workflow.stage.changed
workflow.progress
workflow.completed
workflow.failed
workflow.cancelled
```

## Tool

```text
tool.started
tool.progress
tool.completed
tool.failed
```

## Project

```text
project.index.created
project.index.updated
project.file.changed
project.workspace.stale
```

## Approval

```text
approval.requested
approval.approved
approval.rejected
approval.expired
```

## User Interaction

```text
user.question
user.answer
```

---

# 14. Progress Event

IDE에 진행 상황을 보여주기 위한 표준 Event를 정의한다.

```json
{
  "event": "workflow.progress",

  "data": {
    "stage": "GUIDE_SEARCH",
    "status": "RUNNING",
    "message": "관련 Spring Security 개발 가이드를 검색하고 있습니다."
  }
}
```

IDE에서는 다음처럼 표현할 수 있다.

```text
✓ 요청 분석
✓ 프로젝트 분석
● 개발 가이드 검색
○ 설계
○ 라이브러리 검토
○ 코드 구현
○ 검증
```

---

# 15. Error Model

오류는 자유 텍스트만 반환하지 않고 구조화한다.

```json
{
  "status": "FAILED",

  "error": {
    "code": "WORKSPACE_FILE_NOT_FOUND",

    "category": "WORKSPACE",

    "message": "Requested file does not exist.",

    "retryable": false,

    "details": {
      "path": "src/main/java/AuthService.java"
    }
  }
}
```

---

# 16. Error Category

```text
PROTOCOL

AUTH

PERMISSION

PROJECT

WORKSPACE

TOOL

BUILD

TEST

GIT

DEPENDENCY

SECURITY

GUIDE

CONTEXT

USER

TIMEOUT

SYSTEM
```

---

# 17. 표준 Error Code 예

```text
PROTOCOL_VERSION_UNSUPPORTED
INVALID_REQUEST
INVALID_ARGUMENT

PERMISSION_DENIED
APPROVAL_REQUIRED
APPROVAL_REJECTED

PROJECT_NOT_FOUND

WORKSPACE_FILE_NOT_FOUND
WORKSPACE_FILE_CHANGED
WORKSPACE_STALE

TOOL_NOT_SUPPORTED

BUILD_FAILED
TEST_FAILED

GIT_CONFLICT
GIT_DIRTY_WORKTREE

DEPENDENCY_NOT_FOUND
DEPENDENCY_VULNERABLE

SECURITY_POLICY_VIOLATION

GUIDE_NOT_FOUND

CONTEXT_NOT_FOUND

USER_INPUT_REQUIRED

TIMEOUT

INTERNAL_ERROR
```

---

# 18. Retry 정책

오류에는 retry 가능 여부를 명시한다.

```json
{
  "retryable": true
}
```

다만 자동 retry 여부는 Orchestrator가 결정한다.

추천 기본 정책:

```text
NETWORK_ERROR
TIMEOUT
TEMPORARY_UNAVAILABLE

→ 제한적 retry

PERMISSION_DENIED
SECURITY_POLICY_VIOLATION
APPROVAL_REJECTED

→ retry 금지
```

---

# 19. Tool Namespace

Tool은 domain 단위 Namespace를 사용한다.

```text
project.*

workspace.*

code.*

shell.*

build.*

test.*

git.*

guide.*

dependency.*

security.*

context.*

user.*

workflow.*

system.*
```

---

# 20. project.* Tools

Project Intelligence 및 서버 Project Catalog를 처리한다.

## project.get_profile

```json
{
  "tool": "project.get_profile",
  "arguments": {}
}
```

예상 결과:

```json
{
  "languages": [
    "java",
    "typescript"
  ],

  "frameworks": [
    "spring-boot",
    "react"
  ],

  "build_tools": [
    "maven",
    "npm"
  ],

  "java_version": "17"
}
```

---

## project.get_index

Project Index를 조회한다.

```json
{
  "tool": "project.get_index",
  "arguments": {
    "revision": "latest"
  }
}
```

---

## project.search

서버가 필요한 파일을 탐색할 때 사용하는 핵심 Tool이다.

```json
{
  "tool": "project.search",

  "arguments": {
    "query": "사용자 로그인 인증 및 실패 처리",

    "filters": {
      "languages": [
        "java"
      ],

      "symbol_types": [
        "class",
        "method"
      ]
    },

    "limit": 20
  }
}
```

결과:

```json
{
  "matches": [
    {
      "path": "src/main/java/com/example/AuthService.java",

      "score": 0.94,

      "summary": "사용자 로그인 및 인증 처리",

      "symbols": [
        {
          "name": "login",
          "signature": "login(LoginRequest): LoginResponse"
        }
      ],

      "hash": "sha256:abc123"
    }
  ]
}
```

---

## project.get_symbols

```json
{
  "tool": "project.get_symbols",

  "arguments": {
    "path": "src/main/java/com/example/AuthService.java"
  }
}
```

---

## project.get_dependencies

파일 또는 Symbol의 관계를 조회한다.

```json
{
  "tool": "project.get_dependencies",

  "arguments": {
    "symbol": "AuthService"
  }
}
```

결과:

```json
{
  "uses": [
    "UserRepository",
    "PasswordEncoder"
  ],

  "used_by": [
    "LoginController"
  ]
}
```

---

# 21. workspace.* Tools

Local Workspace 접근용 Tool이다.

## workspace.list

```json
{
  "tool": "workspace.list",

  "arguments": {
    "path": "src/main/java",
    "depth": 2
  }
}
```

---

## workspace.read_file

```json
{
  "tool": "workspace.read_file",

  "arguments": {
    "path": "src/main/java/com/example/AuthService.java"
  }
}
```

---

## workspace.read_files

```json
{
  "tool": "workspace.read_files",

  "arguments": {
    "paths": [
      "AuthService.java",
      "UserRepository.java"
    ]
  }
}
```

---

## workspace.search

텍스트 또는 Regex 기반 로컬 검색을 수행한다.

```json
{
  "tool": "workspace.search",

  "arguments": {
    "query": "login(",
    "scope": "src/main/java"
  }
}
```

---

# 22. workspace.propose_changes

서버가 생성한 변경안을 Local Agent로 전달한다.

파일을 실제로 수정하지 않는다.

```json
{
  "tool": "workspace.propose_changes",

  "arguments": {
    "change_set_id": "CHG-100",

    "changes": [
      {
        "path": "src/main/java/com/example/AuthService.java",

        "operation": "MODIFY",

        "base_hash": "sha256:abc123",

        "content": "..."
      }
    ]
  }
}
```

Local Agent는 현재 파일과 비교하여 Diff를 생성한다.

---

# 23. workspace.apply_changes

개발자 승인이 완료된 change set을 적용한다.

```json
{
  "tool": "workspace.apply_changes",

  "arguments": {
    "change_set_id": "CHG-100",

    "approval_id": "APR-200"
  }
}
```

Local Agent는 적용 직전 다시 hash를 확인한다.

```text
base_hash == current_hash
```

일치하지 않으면:

```text
WORKSPACE_FILE_CHANGED
```

를 반환한다.

---

# 24. Change Operation

지원 operation:

```text
CREATE
MODIFY
DELETE
RENAME
```

Rename:

```json
{
  "operation": "RENAME",

  "path": "OldService.java",

  "target_path": "NewService.java",

  "base_hash": "sha256:..."
}
```

---

# 25. build.* Tools

빌드 도구를 직접 노출하기보다 공통 `build.run`을 우선 권장한다.

```json
{
  "tool": "build.run",

  "arguments": {
    "working_directory": ".",

    "mode": "DEFAULT"
  }
}
```

Local Agent가 Project Profile을 기준으로 Maven/Gradle/npm 등을 선택한다.

필요하면:

```json
{
  "build_tool": "maven"
}
```

을 명시할 수 있다.

---

# 26. Build Result

LLM에게 전체 Build Log를 바로 전달하지 않는다.

구조화된 결과를 반환한다.

```json
{
  "status": "FAILED",

  "build": {
    "exit_code": 1,

    "duration_ms": 8412,

    "errors": [
      {
        "type": "COMPILATION_ERROR",

        "file": "src/main/java/com/example/AuthService.java",

        "line": 84,

        "column": 17,

        "message": "cannot find symbol"
      }
    ],

    "warnings": []
  },

  "log_ref": "artifact://build/EXEC-1001/full.log"
}
```

전체 로그가 필요할 경우 별도로 요청한다.

---

# 27. test.* Tools

기본 인터페이스:

```json
{
  "tool": "test.run",

  "arguments": {
    "scope": "RELATED"
  }
}
```

scope:

```text
RELATED
FILE
MODULE
PROJECT
CUSTOM
```

예:

```json
{
  "scope": "RELATED",

  "changed_files": [
    "AuthService.java"
  ]
}
```

Local Agent가 관련 테스트를 선택할 수 있다.

---

# 28. Test Result

```json
{
  "status": "FAILED",

  "tests": {
    "total": 32,
    "passed": 31,
    "failed": 1,
    "skipped": 0,

    "failures": [
      {
        "test": "AuthServiceTest.loginShouldLockAccount",

        "message": "Expected LOCKED but was ACTIVE",

        "source": {
          "file": "AuthServiceTest.java",
          "line": 83
        }
      }
    ]
  },

  "log_ref": "artifact://test/EXEC-200/full.log"
}
```

---

# 29. shell.* Tools

임의 명령 실행은 위험도가 높으므로 제한적으로 제공한다.

```json
{
  "tool": "shell.execute",

  "arguments": {
    "command": "..."
  }
}
```

실행 전에 반드시 Policy Engine을 거친다.

---

# 30. git.* Tools

## 자동 허용 가능한 Read Tool

```text
git.status
git.diff
git.log
git.show
```

## 상태 변경 Tool

```text
git.checkout
git.add
git.commit
git.merge
git.rebase
git.push
git.reset
git.clean
```

---

## git.status

```json
{
  "tool": "git.status",
  "arguments": {}
}
```

---

## git.commit

```json
{
  "tool": "git.commit",

  "arguments": {
    "message": "feat: add account lock handling",

    "paths": [
      "src/main/java/com/example/AuthService.java"
    ],

    "approval_id": "APR-301"
  }
}
```

---

# 31. guide.* Tools

개발 가이드 검색은 별도 시스템을 호출한다.

## guide.search

```json
{
  "tool": "guide.search",

  "arguments": {
    "query": "로그인 실패 계정 잠금",

    "technology": [
      "java",
      "spring"
    ],

    "categories": [
      "SECURITY",
      "CODING"
    ],

    "limit": 10
  }
}
```

---

## guide.read

```json
{
  "tool": "guide.read",

  "arguments": {
    "guide_id": "GUIDE-87",

    "section_id": "account-lock"
  }
}
```

---

# 32. dependency.* Tools

## dependency.search

Nexus를 조회한다.

```json
{
  "tool": "dependency.search",

  "arguments": {
    "ecosystem": "MAVEN",

    "query": "JWT",

    "constraints": {
      "java_version": "17"
    }
  }
}
```

지원 ecosystem:

```text
MAVEN
NPM
PYPI
```

---

## dependency.resolve

특정 artifact를 프로젝트에서 사용 가능한지 분석한다.

```json
{
  "tool": "dependency.resolve",

  "arguments": {
    "ecosystem": "MAVEN",

    "coordinate": "group:artifact:version"
  }
}
```

---

# 33. security.* Tools

## security.check_dependency

```json
{
  "tool": "security.check_dependency",

  "arguments": {
    "ecosystem": "MAVEN",

    "package": "group:artifact",

    "version": "1.2.3"
  }
}
```

결과:

```json
{
  "allowed": false,

  "findings": [
    {
      "type": "VULNERABILITY",

      "severity": "HIGH",

      "id": "CVE-XXXX-XXXX"
    }
  ]
}
```

---

# 34. context.* Tools

Agent는 Context Storage를 직접 SQL로 접근하지 않는다.

반드시 Context Tool을 사용한다.

## context.get

```json
{
  "tool": "context.get",

  "arguments": {
    "ref": "ctx://work-items/WI-100/design/latest"
  }
}
```

---

## context.put

```json
{
  "tool": "context.put",

  "arguments": {
    "type": "DESIGN",

    "work_item_id": "WI-100",

    "data": {
      "..."
    }
  }
}
```

---

# 35. Context Immutable Versioning

주요 Context는 overwrite보다 version 생성 방식을 권장한다.

```text
DESIGN v1
DESIGN v2
DESIGN v3
```

그리고:

```text
ctx://work-items/WI-100/design/latest
```

로 최신 버전을 조회한다.

과거 버전은 감사 및 반려 분석에 사용할 수 있다.

---

# 36. user.* Tools

Human-in-the-Loop의 핵심 인터페이스이다.

## user.ask

설계 중 사용자 판단이 필요한 경우:

```json
{
  "tool": "user.ask",

  "arguments": {
    "question_id": "Q-100",

    "title": "계정 잠금 해제 정책",

    "message": "로그인 실패 5회 이후 계정 잠금 해제 방법을 선택해 주세요.",

    "type": "SINGLE_SELECT",

    "options": [
      {
        "id": "A",
        "label": "관리자가 직접 잠금 해제"
      },
      {
        "id": "B",
        "label": "30분 후 자동 잠금 해제"
      },
      {
        "id": "C",
        "label": "기존 프로젝트 정책 사용"
      }
    ]
  }
}
```

응답:

```json
{
  "question_id": "Q-100",

  "selected": [
    "C"
  ]
}
```

결과는 Context Storage의 `USER_DECISION`으로 저장한다.

---

# 37. user.request_approval

공통 Approval Tool이다.

```json
{
  "tool": "user.request_approval",

  "arguments": {
    "approval_id": "APR-100",

    "type": "DESIGN",

    "title": "구현 설계 확인",

    "resource_ref": "ctx://work-items/WI-100/design/2",

    "actions": [
      "APPROVE",
      "REQUEST_CHANGE",
      "REJECT"
    ]
  }
}
```

---

# 38. Approval Type

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

# 39. Approval Response

```json
{
  "approval_id": "APR-100",

  "decision": "REJECT",

  "reason": "기존 UserRepository 구조를 사용해서 다시 설계해 주세요."
}
```

반려 사유는 필수로 요구할 수 있다.

특히:

```text
DESIGN
CODE_CHANGE
```

반려에는 reason을 필수로 하는 것을 권장한다.

---

# 40. Local Policy Engine

모든 Local Tool에는 Risk Level을 정의한다.

```text
READ

WRITE

EXECUTE

HIGH_RISK

DESTRUCTIVE
```

예:

```text
workspace.read_file       READ
workspace.search          READ

workspace.apply_changes   WRITE

build.run                 EXECUTE
test.run                  EXECUTE

git.commit                HIGH_RISK
git.push                  HIGH_RISK

git.reset --hard          DESTRUCTIVE
git.clean -fd             DESTRUCTIVE
```

---

# 41. Tool Metadata

각 Tool은 별도의 Definition을 가져야 한다.

예:

```json
{
  "name": "workspace.apply_changes",

  "description": "Apply an approved change set to the local workspace.",

  "execution": "LOCAL",

  "risk": "WRITE",

  "requires_approval": true,

  "idempotent": false,

  "timeout_ms": 30000
}
```

---

# 42. Capability Negotiation

Local Agent의 환경은 서로 다를 수 있다.

연결 시 Capability를 교환한다.

Server:

```json
{
  "tool": "system.get_capabilities",
  "arguments": {}
}
```

Local Agent 응답:

```json
{
  "agent_version": "1.0.0",

  "platform": {
    "os": "WINDOWS",
    "arch": "X86_64"
  },

  "capabilities": {
    "workspace": true,

    "project_index": true,

    "build": [
      "maven",
      "gradle",
      "npm"
    ],

    "test": [
      "junit",
      "pytest",
      "npm"
    ],

    "git": true,

    "languages": [
      "java",
      "python",
      "javascript",
      "typescript"
    ]
  }
}
```

Orchestrator는 이 정보를 기준으로 사용할 Tool을 결정한다.

---

# 43. Tool Discovery

향후 Tool 확장을 고려하여 Discovery 기능을 제공한다.

```text
system.list_tools
```

결과:

```json
{
  "tools": [
    {
      "name": "workspace.read_file",
      "version": "1.0"
    },
    {
      "name": "build.run",
      "version": "1.0"
    }
  ]
}
```

---

# 44. Cancellation

장시간 실행되는 Tool은 취소 가능해야 한다.

```json
{
  "protocol": "oncode-tool/1.0",
  "message_type": "CANCEL",

  "payload": {
    "execution_id": "EXEC-100"
  }
}
```

결과:

```text
tool.cancelled
```

Event를 발생시킨다.

---

# 45. Timeout

Request option:

```json
{
  "options": {
    "timeout_ms": 30000
  }
}
```

Timeout 발생:

```json
{
  "status": "TIMEOUT",

  "error": {
    "code": "TIMEOUT",
    "retryable": true
  }
}
```

---

# 46. Streaming

소스 파일이나 Build Log처럼 응답이 클 수 있는 경우 Chunk Streaming을 지원할 수 있다.

다만 v1 초기 구현에서는 모든 Tool에 Streaming을 요구하지 않는다.

큰 데이터는 가능한 경우 `artifact_ref`를 사용한다.

예:

```json
{
  "result": {
    "log_ref": "artifact://build/EXEC-100/full.log"
  }
}
```

필요한 부분만 별도 Tool로 조회한다.

이 방식이 LLM Context 관리 측면에서도 유리하다.

---

# 47. Idempotency

일부 작업은 재전송에 주의해야 한다.

Request에는 선택적으로 다음을 지정할 수 있다.

```json
{
  "options": {
    "idempotency_key": "IDEMP-100"
  }
}
```

특히 다음 작업에 중요하다.

```text
workspace.apply_changes
git.commit
git.push
dependency.install
```

네트워크 오류로 Request가 재전송되어도 동일 작업이 중복 실행되지 않도록 한다.

---

# 48. Correlation

모든 Event와 Response는 원래 Tool Call과 연결되어야 한다.

```text
call_id
execution_id
```

두 식별자를 사용한다.

```text
CALL-100
   ↓
EXEC-300
   ├─ started
   ├─ progress
   └─ completed
```

---

# 49. Security

프로토콜 계층에서는 다음을 기본 원칙으로 한다.

1. 모든 연결 인증
2. Session 및 Project Scope 검증
3. Tool별 권한 검증
4. Local Tool Policy 적용
5. 경로 traversal 차단
6. Workspace root 밖 접근 차단
7. Shell command 정책 검증
8. Approval 위조 방지
9. Audit Log 저장
10. Secret masking

---

# 50. Workspace Boundary

다음과 같은 요청은 반드시 차단한다.

```text
../../Windows/System32
/etc/passwd
사용자 홈의 다른 프로젝트
```

Local Agent는 프로젝트 Root를 기준으로 sandbox를 구성한다.

```text
workspace_root
   ↓
허용 영역
```

기본적으로 해당 영역 밖의 File Tool 접근은 허용하지 않는다.

---

# 51. Secret Handling

Local Agent가 파일을 Server로 전달하기 전 Secret Scanner를 수행할 수 있도록 설계하는 것을 권장한다.

예:

```text
.env
credential
private key
access token
password
```

발견 시:

```text
MASK

DENY

USER_APPROVAL
```

중 정책을 적용할 수 있다.

---

# 52. PROJECT_INDEX 동기화

Local Agent Project Intelligence가 index를 생성하면 Server에 등록한다.

```text
project.index.created
```

이후 변경:

```text
project.index.updated
```

Event를 전달한다.

전체 index를 매번 전송하지 않고 revision 기반 delta sync를 지원하는 것이 바람직하다.

예:

```json
{
  "event": "project.index.updated",

  "data": {
    "revision": 103,

    "previous_revision": 102,

    "changed": [
      "AuthService.java"
    ],

    "deleted": []
  }
}
```

---

# 53. Workspace Revision

Project Index 전체에 revision을 부여한다.

```text
project_revision = 103
```

Server가 작업을 설계할 때:

```text
base_project_revision = 103
```

을 저장한다.

작업 적용 시 현재 revision이 크게 변경되었다면 재검토할 수 있다.

---

# 54. Stale Context 방지

파일 단위로는 hash를 사용한다.

프로젝트 단위로는 revision을 사용한다.

```text
Project Revision
+
File Hash
```

두 체계를 병행한다.

예:

```text
설계 기준 revision = 103
현재 revision = 115

AuthService hash도 변경
```

→ `STALE_WORKSPACE`

→ 다시 관련 코드 분석

이 흐름을 권장한다.

---

# 55. Tool Call Audit

Tool Call은 최소 다음 정보를 기록한다.

```text
call_id

actor

tool

arguments hash

start_time

end_time

status

approval_id

result summary

error

project_id

workflow_id

work_item_id
```

소스코드 전체나 비밀정보가 Audit Log에 그대로 기록되지 않도록 별도의 masking 정책을 적용한다.

---

# 56. Tool Protocol과 Workflow의 관계

Tool Protocol이 Workflow를 결정하지 않는다.

```text
Workflow Engine
      ↓
현재 State 결정
      ↓
Agent 실행
      ↓
Agent Tool Calling
      ↓
Tool Protocol
```

즉:

> Tool Protocol은 "어떻게 호출하는가"를 정의하고  
> Workflow는 "언제 무엇을 호출하는가"를 정의한다.

두 영역은 분리한다.

---

# 57. Agent 간 호출

서버 Agent 간 직접 Agent-to-Agent Prompt 전달은 사용하지 않는다.

예를 들어 Senior Developer가 Guide Agent를 사용해야 한다면 논리적으로는:

```text
Senior Developer
      ↓
Orchestrator
      ↓
Guide Agent Task
```

가 되고, Guide Agent 결과는 Context Storage에 기록한다.

```text
Guide Agent
      ↓
context.put
      ↓
Guide Result
```

Senior Developer는:

```text
context.get
```

을 사용해서 읽는다.

따라서 Tool Protocol의 `actor`에는 Agent가 나타날 수 있지만 Agent 자체의 private context는 Protocol Payload로 전달하지 않는다.

---

# 58. Request Context Reference

Tool 실행에 필요한 Context를 명시적으로 참조할 수 있다.

```json
{
  "context_refs": [
    {
      "type": "REQUIREMENT",
      "ref": "ctx://work-items/WI-100/requirement/latest"
    },

    {
      "type": "DESIGN",
      "ref": "ctx://work-items/WI-100/design/2"
    }
  ]
}
```

Tool이나 Agent runtime은 필요한 항목만 가져온다.

---

# 59. 초기 Tool Set

onCode MVP에서는 다음 Tool부터 구현하는 것을 권장한다.

### Project

```text
project.get_profile
project.get_index
project.search
project.get_symbols
project.get_dependencies
```

### Workspace

```text
workspace.list
workspace.read_file
workspace.read_files
workspace.search
workspace.propose_changes
workspace.apply_changes
```

### Build/Test

```text
build.run
test.run
```

### Git

```text
git.status
git.diff
git.log
git.branch
git.checkout
git.add
git.commit
```

초기 MVP에서는 `git.push`, `reset`, `clean`, `rebase`는 후순위로 두어도 된다.

### Knowledge

```text
guide.search
guide.read

dependency.search
dependency.resolve

security.check_dependency
```

### Context

```text
context.get
context.put
```

### Human-in-the-Loop

```text
user.ask
user.request_approval
```

### System

```text
system.get_capabilities
system.list_tools
system.ping
```

---

# 60. 대표 구현 Workflow

사용자 요청:

```text
"로그인 실패가 5회 이상이면 계정을 잠그도록 수정해줘."
```

Tool 흐름:

```text
project.search
      ↓
workspace.read_files
      ↓
guide.search
      ↓
guide.read
      ↓
context.put(requirement)
      ↓
context.put(design)
      ↓
user.request_approval(DESIGN)
      ↓
dependency.resolve
      ↓
security.check_dependency
      ↓
workspace.propose_changes
      ↓
user.request_approval(CODE_CHANGE)
      ↓
workspace.apply_changes
      ↓
build.run
      ↓
test.run
```

실패 시:

```text
build.run
   ↓
BUILD_FAILED
   ↓
context.put(build_result)
   ↓
Senior Developer 재분석
   ↓
workspace.read_files
   ↓
workspace.propose_changes
```

---

# 61. 프로토콜 계층 구조

전체적으로 다음 4계층으로 보는 것이 적절하다.

```text
┌──────────────────────────────┐
│ Workflow / Orchestration     │
├──────────────────────────────┤
│ Agent Runtime                │
├──────────────────────────────┤
│ onCode Tool Protocol         │
├──────────────────────────────┤
│ HTTP / WS / gRPC / MQ        │
└──────────────────────────────┘
```

Tool Protocol은 Agent Framework나 Transport와 독립적이어야 한다.

---

# 62. 향후 MCP 연계

내부 Tool Protocol과 MCP를 동일하게 만들 필요는 없다.

권장 구조:

```text
onCode Agent
     ↓
onCode Tool Protocol
     ↓
Tool Registry
     │
     ├─ Native Tool
     │
     ├─ Local Agent Tool
     │
     └─ MCP Adapter
              ↓
           MCP Server
```

즉 onCode 내부 표준은 자체적으로 유지하고 외부 Tool을 MCP Adapter로 연결한다.

이 구조가 폐쇄망 및 정책 통제 측면에서 더 유리하다.

---

# 63. v1에서 명확하게 제외할 항목

Tool Protocol v1 자체에서는 다음을 정의하지 않는다.

```text
Agent Prompt

Agent 내부 Reasoning

LLM Model 선택 정책

Workflow 상세 State Transition

개발 가이드 검색 알고리즘

Nexus 내부 구현

Project Summary 생성 알고리즘

Security DB 구축 방법

IDE UI 디자인
```

이 항목들은 각각 별도 명세에서 정의한다.

---

# 64. onCode Tool Protocol v1 핵심 원칙 요약

onCode Tool Protocol의 핵심 원칙은 다음과 같다.

1. **JSON 기반 Tool Calling**
2. **Transport 독립적**
3. **Server/Local Tool 동일 모델**
4. **Agent 간 직접 Context 전달 금지**
5. **Context Storage Reference 기반 협업**
6. **모든 작업에 Correlation ID 적용**
7. **구조화된 Error Model**
8. **비동기 Execution/Event 지원**
9. **Human Approval을 Protocol의 1급 객체로 처리**
10. **파일 변경 전 Proposal과 실제 Apply 분리**
11. **File Hash 및 Project Revision 기반 Stale Context 방지**
12. **Local Policy Engine을 통한 위험 작업 통제**
13. **Project Index 기반 최소 소스 요청**
14. **Audit 가능한 Tool Execution**
15. **향후 MCP Adapter 확장 가능 구조**

---

# 65. 권장 후속 명세

Tool Protocol v1 이후에는 다음 순서로 상세 명세를 작성하는 것이 적절하다.

```text
SPEC-01
onCode Tool Protocol v1
        ↓
SPEC-02
Workflow State Machine
        ↓
SPEC-03
Context Storage Data Model / ERD
        ↓
SPEC-04
Local Agent Architecture
        ↓
SPEC-05
Project Intelligence / PROJECT_INDEX Schema
        ↓
SPEC-06
Server Multi-Agent Architecture
        ↓
SPEC-07
IDE Extension ↔ Local Agent Protocol
        ↓
SPEC-08
Security / Local Policy Model
```

이 순서로 진행하면 각 명세 간 의존관계가 비교적 깔끔하게 유지된다.