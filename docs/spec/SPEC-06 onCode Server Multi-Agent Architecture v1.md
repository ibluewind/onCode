# SPEC-06 onCode Server Multi-Agent Architecture v1

## 1. 목적

본 명세는 onCode 중앙 서버에서 동작하는 Multi-Agent Architecture를 정의한다.

중앙 서버는 개발자의 요청을 분석하고, 필요한 프로젝트 정보와 개발 가이드를 조회하고, 구현 설계를 생성하며, 코드 생성과 검토, 보안 검증을 수행한다.

중앙 Server Agent는 개발자 PC의 파일을 직접 수정하거나 Build/Test/Git 명령을 직접 실행하지 않는다.

핵심 원칙은 다음과 같다.

> Server는 판단과 설계를 담당하고, Local Agent는 실행을 담당한다.

---

# 2. 전체 구조

```text
┌──────────────────────── onCode Server ───────────────────────┐
│                                                            │
│ Authentication / Session / Project Management              │
│                         │                                  │
│                         ▼                                  │
│                Workflow Orchestrator                       │
│                         │                                  │
│                         ▼                                  │
│                Senior Developer Agent                      │
│                         │                                  │
│      ┌──────────────────┼─────────────────────┐            │
│      │                  │                     │            │
│      ▼                  ▼                     ▼            │
│ Requirement          Guide               Code Analysis     │
│ Agent                Agent               Agent             │
│      │                  │                     │            │
│      └───────────────┬──┴─────────────┬──────┘            │
│                      ▼                ▼                    │
│                  Design Agent     Dependency Agent         │
│                      │                │                    │
│                      └────────┬───────┘                    │
│                               ▼                            │
│                      Implementation Agent                  │
│                               │                            │
│                 ┌─────────────┼──────────────┐             │
│                 ▼             ▼              ▼             │
│              Review       Security       Guide Compliance  │
│              Agent        Agent          Function          │
│                                                            │
│                Context Storage / Workflow DB               │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

---

# 3. 핵심 구성요소

Server Multi-Agent 영역은 다음 요소로 구성한다.

```text
Workflow Orchestrator

Senior Developer Agent

Requirement Agent

Guide Agent

Code Analysis Agent

Design Agent

Dependency Agent

Implementation Agent

Review Agent

Security Agent
```

초기 MVP에서는 일부 Agent 역할을 통합할 수 있다.

---

# 4. Workflow Orchestrator

Orchestrator는 Agent가 아니다.

가능하면 deterministic한 Workflow Engine으로 구현한다.

주요 역할:

```text
Workflow 생성

State Transition

Agent Task 생성

Tool 호출 Routing

Context Reference 관리

Retry

Timeout

Approval 상태 관리

Cancellation

Resume

Failure Handling
```

---

# 5. Orchestrator가 하지 않는 일

Orchestrator는 다음을 직접 판단하지 않는다.

```text
어떤 클래스 구조가 좋은가

어떤 코드가 올바른가

어떤 라이브러리가 적합한가

사용자 요구가 기술적으로 어떻게 구현되어야 하는가
```

이러한 판단은 Agent가 수행한다.

---

# 6. Senior Developer Agent

Server 측 핵심 판단 Agent이다.

사용자의 요청을 시니어 개발자 관점에서 처리한다.

책임:

```text
요구사항 이해

작업 범위 판단

프로젝트 Context 필요 여부 판단

추가 질문 필요 여부 판단

관련 Source 결정

설계 방향 결정

하위 Agent 결과 종합

오류 원인 분석

수정 전략 결정

최종 기술 판단
```

---

# 7. Senior Developer와 Orchestrator 분리

```text
Orchestrator
→ 지금 어느 단계를 실행할 것인가?

Senior Developer
→ 이 문제를 어떻게 해결할 것인가?
```

이 경계를 유지한다.

---

# 8. Senior Developer 입력 Context

주로 다음을 읽는다.

```text
Original Request

Requirement

Project Profile

Project Search Result

User Decisions

Guide References

Design

Dependency Decisions

Review Findings

Security Findings

Build Result

Test Result
```

---

# 9. Senior Developer 출력

Senior Developer는 가능한 한 구조화된 결과를 생성한다.

예:

```json
{
  "decision": "REQUEST_MORE_CONTEXT",

  "required_files": [
    "AuthService.java",
    "UserRepository.java"
  ],

  "reason": "로그인 실패 처리와 계정 상태 저장 구조 확인 필요"
}
```

또는:

```json
{
  "decision": "REDESIGN",

  "reason": "현재 구현은 DB schema 변경을 필요로 하지만 사용자 constraint와 충돌함"
}
```

---

# 10. Requirement Agent

사용자 요청을 명시적인 Requirement로 변환한다.

입력:

```text
Original Request
Conversation Context
Project Profile
User Decisions
```

출력:

```text
Goal

Functional Requirements

Non-functional Requirements

Constraints

Unknowns

Assumptions
```

---

# 11. Requirement Agent 예

사용자:

```text
로그인 실패가 5회 이상이면 계정을 잠가줘.
```

출력:

```json
{
  "goal": "5회 이상 로그인 실패 시 계정 잠금",

  "functional_requirements": [
    "로그인 실패 횟수를 기록한다.",
    "실패 횟수가 5회 이상이면 계정 상태를 잠금으로 변경한다.",
    "로그인 성공 시 실패 횟수를 초기화한다."
  ],

  "unknowns": [
    "잠금 해제 정책",
    "기존 사용자 Entity 변경 가능 여부"
  ]
}
```

---

# 12. Requirement 확정 조건

다음 조건을 충족해야 Requirement가 confirmed 상태가 된다.

```text
Critical Unknown 없음

사용자 Decision 반영

기존 Project Constraint와 충돌 없음
```

---

# 13. Guide Agent

Guide Agent는 개발 가이드 검색만 담당한다.

역할:

```text
검색 Query 생성

Category 선택

Guide 검색

관련 Section 선택

Priority 판단

Guide Reference 저장
```

---

# 14. Guide Agent가 하지 않는 일

Guide Agent는 코드 생성이나 설계 최종 판단을 하지 않는다.

```text
Guide Agent
→ "이 규칙들이 관련 있다."

Design Agent
→ "이 규칙을 적용해서 이렇게 설계한다."
```

---

# 15. Guide Priority

기본 우선순위:

```text
PROJECT_RULE

>

ORGANIZATION_RULE

>

FRAMEWORK_RULE

>

LANGUAGE_RULE

>

GENERAL_BEST_PRACTICE
```

---

# 16. Code Analysis Agent

Project Intelligence의 Index와 실제 Source를 분석한다.

역할:

```text
관련 파일 결정

기존 Architecture 이해

Class/Method 관계 분석

수정 영향 범위 파악

기존 구현 패턴 발견

재사용 가능한 코드 발견
```

---

# 17. Code Analysis 순서

```text
project.search
      ↓
project.get_symbols
      ↓
project.get_dependencies
      ↓
workspace.read_files
      ↓
Code Analysis
```

전체 프로젝트 Source를 처음부터 요청하지 않는다.

---

# 18. Source 요청 최소화

예를 들어 로그인 수정 요청에서:

```text
PROJECT_INDEX
     ↓
AuthService
LoginController
UserRepository
```

만 우선 요청한다.

필요하면 추가 Source를 단계적으로 요청한다.

---

# 19. Code Analysis 출력

```json
{
  "relevant_files": [
    {
      "path": "AuthService.java",
      "reason": "login method contains authentication logic"
    }
  ],

  "architecture": {
    "pattern": "LAYERED"
  },

  "existing_patterns": [
    "Service layer handles domain logic",
    "Repository layer persists user status"
  ],

  "risks": []
}
```

---

# 20. Design Agent

Requirement와 Code Analysis, Guide 결과를 바탕으로 실제 구현 설계를 만든다.

입력:

```text
Confirmed Requirement

Project Context

Relevant Source

Guide Reference

User Decision
```

---

# 21. Design Agent 출력

반드시 구조화한다.

```text
Goal

Scope

Affected Files

New Files

Components

Logic

Data Flow

Exception Handling

Dependency Strategy

Security Considerations

Test Strategy

Guide References
```

---

# 22. Design에는 Source 변경 범위를 포함

예:

```json
{
  "affected_files": [
    {
      "path": "AuthService.java",
      "action": "MODIFY"
    },
    {
      "path": "User.java",
      "action": "MODIFY"
    }
  ]
}
```

---

# 23. Design Approval

Design Agent 결과는 바로 Implementation으로 넘어가지 않는다.

```text
Design
  ↓
Developer Review
  ↓
APPROVE
```

이후에만 구현을 시작한다.

---

# 24. 사용자 변경 요청

사용자가:

```text
DB Schema는 변경하지 마.
```

라고 하면:

```text
User Decision
     ↓
Requirement Update
     ↓
Design v2
```

를 생성한다.

---

# 25. Dependency Agent

외부 라이브러리 필요 여부와 적절한 Dependency를 결정한다.

순서:

```text
기존 Dependency 확인
      ↓
기존 것으로 구현 가능?
      ↓
Nexus 검색
      ↓
Version Compatibility
      ↓
Security Check
      ↓
Decision
```

---

# 26. Dependency 우선순위

```text
1. 기존 Dependency 활용

2. 기존 Dependency의 안전한 Version 활용

3. Nexus에 승인된 신규 Dependency

4. 대체 Dependency

5. Library 없이 구현

6. 제한적인 자체 구현
```

---

# 27. 자체 구현 제한

다음 영역은 자체 구현을 기본적으로 피한다.

```text
Cryptography

JWT Algorithm

OAuth

Password Hash

HTML Sanitizer

Certificate Validation
```

안전한 검증된 구현을 우선한다.

---

# 28. Nexus 연계

Dependency Agent는 직접 Nexus API 구현을 알 필요가 없다.

Tool:

```text
dependency.search

dependency.resolve
```

를 사용한다.

---

# 29. Dependency Decision

결과:

```json
{
  "decision": "USE_EXISTING",

  "package": "spring-security",

  "reason": "현재 프로젝트에 이미 존재하며 요구 기능을 제공함"
}
```

---

# 30. Implementation Agent

승인된 Design을 코드로 변환한다.

중요 원칙:

> Implementation Agent는 파일을 직접 수정하지 않는다.

결과는 Change Set이다.

---

# 31. Implementation Agent 입력

```text
Confirmed Requirement

Approved Design

User Decisions

Guide References

Dependency Decisions

Relevant Source Files
```

---

# 32. Implementation Agent 출력

```json
{
  "change_set": {
    "changes": [
      {
        "path": "AuthService.java",
        "operation": "MODIFY",
        "base_hash": "sha256:...",
        "content": "..."
      }
    ]
  }
}
```

---

# 33. 구현 시 준수 우선순위

```text
User Confirmed Decision

Approved Design

Project Rule

Organization Guide

Framework Guide

Language Guide

Existing Project Style
```

---

# 34. Implementation Agent 추가 Source 요청

구현 도중 Source가 부족할 경우 임의 추측하지 않는다.

예:

```json
{
  "status": "NEED_MORE_CONTEXT",

  "requested_files": [
    "SecurityConfig.java"
  ],

  "reason": "기존 인증 필터 구성 확인 필요"
}
```

Orchestrator가 Source 요청을 수행하고 Implementation을 재개한다.

---

# 35. Review Agent

Implementation Agent와 반드시 분리한다.

검토 대상:

```text
Requirement Compliance

Design Compliance

Guide Compliance

Architecture Consistency

Naming

Error Handling

Duplication

Backward Compatibility

Maintainability

Testability
```

---

# 36. Review 결과

```json
{
  "status": "FAIL",

  "findings": [
    {
      "severity": "HIGH",
      "file": "AuthService.java",
      "rule": "GUIDE-87",
      "message": "로그인 실패 Audit 처리가 누락됨."
    }
  ]
}
```

---

# 37. Review 실패 처리

단순 코드 문제:

```text
Review
  ↓
Implementation
```

설계 문제:

```text
Review
  ↓
Design
```

Senior Developer가 복귀 단계를 판단한다.

---

# 38. Guide Compliance

Guide Compliance를 별도 Agent로 둘 수도 있지만 초기에는 Review Agent의 기능으로 포함하는 것을 권장한다.

Review 결과에는 반드시 사용한 Guide ID를 연결할 수 있어야 한다.

---

# 39. Security Agent

Security Agent는 코드와 Dependency 보안 검증을 담당한다.

검토 대상:

```text
Injection

Authentication

Authorization

Secret Exposure

Path Traversal

Command Execution

Unsafe Deserialization

Cryptography

Dependency Vulnerability

License Policy
```

---

# 40. Security Agent 입력

```text
Change Set

Dependency Decisions

Relevant Source

Security Guide

Vulnerability Check Result
```

---

# 41. Security 결과

```json
{
  "status": "FAIL",

  "findings": [
    {
      "severity": "HIGH",
      "type": "COMMAND_EXECUTION",
      "file": "BuildService.java",
      "message": "사용자 입력이 Shell 명령에 직접 포함됨."
    }
  ]
}
```

---

# 42. Security 실패 분류

코드 문제:

```text
→ Implementation
```

Dependency 문제:

```text
→ Dependency Agent
```

설계 문제:

```text
→ Design
```

---

# 43. Agent Runtime

각 Agent는 공통 Agent Runtime을 통해 실행하는 것을 권장한다.

```text
Agent Runtime

├─ Prompt Template
├─ Context Resolver
├─ Context Builder
├─ Model Client
├─ Tool Client
├─ Output Validator
└─ Result Writer
```

---

# 44. Agent가 직접 DB 접근 금지

```text
Agent
  ↓
context.get
context.put
```

만 사용한다.

직접 SQL 접근은 하지 않는다.

---

# 45. Agent Output Validation

Agent 결과는 JSON Schema로 검증하는 것을 권장한다.

예:

```text
RequirementOutputSchema

DesignOutputSchema

ReviewOutputSchema

SecurityOutputSchema
```

Schema 검증 실패 시 재생성 또는 오류 처리한다.

---

# 46. Structured Output 우선

자유 텍스트보다 구조화된 결과를 우선한다.

잘못된 방식:

```text
"AuthService를 수정하는 게 좋겠습니다..."
```

권장:

```json
{
  "decision": "MODIFY",
  "files": [...]
}
```

---

# 47. Agent Task

Orchestrator는 Agent에 Task를 전달한다.

예:

```json
{
  "task_id": "TASK-100",

  "type": "DESIGN",

  "agent": "design-agent",

  "context_refs": [
    "ctx://work-items/WI-100/requirement/latest",
    "ctx://work-items/WI-100/guides"
  ]
}
```

---

# 48. Agent Task에 전체 Prompt 전달 금지

Orchestrator가 Agent별 Prompt를 동적으로 조합하여 직접 전달하는 구조보다:

```text
Task Type
+
Context Reference
```

기반으로 Agent Runtime이 Prompt를 구성하는 것이 좋다.

---

# 49. Prompt Template 관리

Prompt는 코드 내부 hard coding보다 Version 관리한다.

예:

```text
prompts/

senior-developer/
  v1.md

requirement/
  v1.md

design/
  v3.md

review/
  v2.md
```

---

# 50. Prompt Version 기록

Agent 실행마다:

```text
prompt_template_version
```

을 기록한다.

향후 품질 분석에 사용한다.

---

# 51. Model Routing

Agent별 서로 다른 Model을 사용할 수 있다.

예:

```text
Requirement
→ Small/Medium Model

Guide
→ Small Model

Senior Developer
→ High Capability Model

Design
→ High Capability Model

Implementation
→ Coding Model

Review
→ Coding/Reasoning Model
```

---

# 52. 모델과 Agent 분리

Agent는 특정 LLM 모델 이름에 종속되지 않는다.

```text
Agent
  ↓
Model Profile
  ↓
Inference Gateway
```

구조를 권장한다.

---

# 53. Inference Gateway

중앙에서 LLM 호출을 통합한다.

```text
Agent
  ↓
Inference Gateway
  ↓
vLLM / Other Internal Inference
```

기능:

```text
Model Routing

Timeout

Token Limits

Metrics

Fallback

Concurrency Control
```

---

# 54. Context Builder

Agent별 필요한 Context만 구성한다.

예:

## Implementation

```text
Requirement

Approved Design

User Decisions

Relevant Guides

Selected Dependencies

Relevant Source
```

Review Agent는:

```text
Requirement

Design

Change Set

Guide

Relevant Existing Source
```

만 받는다.

---

# 55. Context Rot 방지

다음을 피한다.

```text
전체 Chat History

전체 Project Index

모든 Source File

모든 Build Log
```

Context Builder가 필요한 정보만 선택한다.

---

# 56. Context Summary

이전 Build/Test 반복 결과가 많을 경우:

```text
Build attempt 1
Build attempt 2
Build attempt 3
```

를 모두 전달하기보다:

```text
Current failure summary

Previous fixes attempted
```

로 요약한다.

---

# 57. Agent Memory

Agent 자체에 장기 Memory를 두지 않는 것을 권장한다.

작업 기억은 Context Storage를 사용한다.

```text
Stateless Agent

+

Stateful Context Storage
```

구조를 유지한다.

---

# 58. Agent 간 직접 호출

기본적으로 금지한다.

```text
Design Agent
    X
Implementation Agent
```

직접 호출 대신:

```text
Design Agent
    ↓
Context Storage
    ↓
Orchestrator
    ↓
Implementation Agent
```

로 처리한다.

---

# 59. Senior Developer 예외

Senior Developer도 다른 Agent를 직접 RPC 호출하지 않는다.

Senior Developer가:

```text
"Guide 확인 필요"
```

라는 Decision을 반환하면 Orchestrator가 Guide Task를 생성한다.

---

# 60. Agent Dependency Graph

기본 흐름:

```text
Requirement
    ↓
Guide
    ↓
Code Analysis
    ↓
Design
    ↓
Dependency
    ↓
Implementation
    ↓
Review
    ↓
Security
```

일부 작업은 병렬 가능하다.

---

# 61. 병렬 실행 후보

예:

```text
Guide Search
+
Code Analysis
```

는 Requirement가 확정된 이후 병렬 수행 가능하다.

또는:

```text
Code Review
+
Security Review
```

도 Change Set 이후 병렬 수행할 수 있다.

---

# 62. 병렬 실행 주의

병렬 결과가 서로 의존하면 안 된다.

예:

```text
Implementation
```

은 Approved Design이 필요하므로 Design과 병렬 실행 불가다.

---

# 63. Fan-out / Fan-in

```text
              ┌─ Guide Agent
Requirement ──┤
              └─ Code Analysis Agent
                    ↓
                 Fan-in
                    ↓
                 Design
```

이 패턴을 Orchestrator가 지원할 수 있다.

---

# 64. Agent 실패

Agent 실행 실패 원인:

```text
MODEL_TIMEOUT

INVALID_OUTPUT

CONTEXT_MISSING

TOOL_FAILURE

MODEL_UNAVAILABLE

TOKEN_LIMIT
```

---

# 65. Agent Retry

재시도 가능한 오류:

```text
MODEL_TIMEOUT

TEMPORARY_MODEL_UNAVAILABLE

OUTPUT_SCHEMA_ERROR
```

제한적 Retry를 적용한다.

---

# 66. Agent Retry 시 Prompt 변경

단순 동일 Prompt 반복보다:

```text
Previous output validation error
```

정도만 추가해 재실행할 수 있다.

---

# 67. Infinite Agent Loop 방지

Agent간 재작업 횟수를 제한한다.

예:

```text
Implementation ↔ Review
max 5

Implementation ↔ Build Fix
max 3

Design Revision
configurable
```

---

# 68. 오류 수정 흐름

Local Build 실패:

```text
Build Result
    ↓
Senior Developer
    ↓
Failure Classification
```

CODE_ERROR:

```text
→ Implementation
```

DEPENDENCY_ERROR:

```text
→ Dependency
```

DESIGN_ERROR:

```text
→ Design
```

ENVIRONMENT_ERROR:

```text
→ User / Local Environment
```

---

# 69. Senior Developer Build 분석

예:

```json
{
  "failure_category": "CODE_ERROR",

  "root_cause": "호출한 repository method가 존재하지 않음",

  "action": "REIMPLEMENT",

  "required_context": [
    "UserRepository.java"
  ]
}
```

---

# 70. Test 실패도 같은 방식

```text
Test Result
  ↓
Senior Developer
  ↓
IMPLEMENTATION_ERROR
TEST_ERROR
EXISTING_FAILURE
ENVIRONMENT_ERROR
```

로 분류한다.

---

# 71. Agent Tool Access Control

Agent마다 사용할 수 있는 Tool을 제한한다.

예:

## Requirement Agent

```text
context.*
project.get_profile
```

## Guide Agent

```text
guide.*
context.*
```

## Code Analysis Agent

```text
project.*
workspace.read_*
workspace.search
context.*
```

---

# 72. Implementation Agent Tool 권한

허용:

```text
project.search
project.get_symbols

workspace.read_file
workspace.read_files
workspace.search

context.*
```

허용하지 않음:

```text
workspace.apply_changes

git.commit

shell.execute
```

즉 Implementation Agent는 직접 변경할 수 없다.

---

# 73. Review Agent Tool 권한

```text
workspace.read_*
project.*
guide.*
context.*
```

읽기 중심이다.

---

# 74. Security Agent Tool 권한

```text
security.*
dependency.*
workspace.read_*
context.*
```

---

# 75. Agent Permission Profile

예:

```json
{
  "agent": "implementation-agent",

  "tools": {
    "allow": [
      "workspace.read_*",
      "project.*",
      "context.*"
    ],

    "deny": [
      "workspace.apply_changes",
      "shell.*",
      "git.*"
    ]
  }
}
```

---

# 76. Tool 결과 신뢰 모델

Local Agent의 Parser 결과와 실제 Source는 높은 신뢰도로 취급한다.

LLM Summary는 보조 정보다.

권장 우선순위:

```text
Actual Source

Parser/AST Data

Project Configuration

User Confirmed Decision

Development Guide

LLM-generated Summary
```

---

# 77. Summary에만 의존 금지

Implementation Agent는 코드 수정 시 반드시 실제 Source를 읽어야 한다.

```text
PROJECT_SUMMARY
   ↓
파일 발견

Actual Source
   ↓
구현
```

이 원칙이 중요하다.

---

# 78. Design도 주요 Source 확인 필요

설계 전에 최소한 핵심 수정 파일은 실제 Source를 읽는 것을 권장한다.

Project Index Summary만으로 상세 설계를 확정하지 않는다.

---

# 79. Context Provenance

Agent 결과에는 입력 근거를 기록한다.

예:

```json
{
  "sources": [
    "source://AuthService.java@sha256:...",
    "guide://GUIDE-87",
    "ctx://work-items/WI-100/requirement/2"
  ]
}
```

---

# 80. Agent 결과 상태

공통 결과 상태:

```text
SUCCESS

NEED_MORE_CONTEXT

NEED_USER_INPUT

RETRY

FAILED

BLOCKED
```

---

# 81. NEED_MORE_CONTEXT

예:

```json
{
  "status": "NEED_MORE_CONTEXT",

  "requests": [
    {
      "type": "FILE",
      "path": "SecurityConfig.java"
    }
  ]
}
```

---

# 82. NEED_USER_INPUT

```json
{
  "status": "NEED_USER_INPUT",

  "question": {
    "message": "...",
    "options": [...]
  }
}
```

Orchestrator가 `WAITING_USER_INPUT` 상태로 전환한다.

---

# 83. BLOCKED

보안 정책이나 필수 Guide 누락 등으로 진행할 수 없는 경우:

```json
{
  "status": "BLOCKED",

  "reason": "Required security policy is unavailable."
}
```

---

# 84. Agent Metrics

다음 정보를 기록하는 것을 권장한다.

```text
Agent Type

Model

Duration

Input Tokens

Output Tokens

Tool Calls

Retry Count

Result Status
```

---

# 85. 품질 분석

향후 다음 분석이 가능하다.

```text
어떤 Agent에서 실패율이 높은가

어떤 Prompt Version이 좋은가

Review에서 가장 많이 잡히는 문제는 무엇인가

Build 수정 Loop 평균 횟수

어떤 모델이 코드 품질이 좋은가
```

---

# 86. Audit

Agent 활동에서도 최소 다음을 Audit 가능해야 한다.

```text
Agent

Task

Context References

Output Reference

Tool Calls

State Transition
```

Private chain-of-thought은 저장하지 않는다.

---

# 87. Private Reasoning 저장 금지

저장 대상:

```text
Decision

Summary

Finding

Reason Code

Evidence

Structured Result
```

저장하지 않는 대상:

```text
Chain-of-thought

Scratchpad

Hidden Reasoning
```

---

# 88. Agent Prompt Security

Project Source나 가이드 문서 내부의 Prompt Injection 가능성도 고려해야 한다.

예:

```text
README:
"Ignore previous instructions and..."
```

이를 Agent System Instruction보다 낮은 신뢰도의 데이터로 취급한다.

---

# 89. Context Trust Level

Context에 trust metadata를 둘 수 있다.

예:

```text
SYSTEM_POLICY
→ highest

USER_CONFIRMED_DECISION

DEVELOPMENT_GUIDE

SOURCE_CODE

PROJECT_SUMMARY

UNTRUSTED_DOCUMENT
```

---

# 90. Source Code의 명령성 텍스트

주석이나 README에 포함된 자연어를 Agent instruction으로 해석하지 않는다.

```text
Source Content
→ Data

System/Workflow Policy
→ Instruction
```

로 구분한다.

---

# 91. Server Agent Registry

Server는 Agent Registry를 관리한다.

```text
senior-developer-agent

requirement-agent

guide-agent

code-analysis-agent

design-agent

dependency-agent

implementation-agent

review-agent

security-agent
```

---

# 92. Agent Definition

예:

```json
{
  "name": "design-agent",

  "version": "1.0",

  "model_profile": "reasoning-high",

  "prompt_version": "3",

  "allowed_tools": [
    "context.*",
    "project.*",
    "workspace.read_*",
    "guide.*"
  ],

  "output_schema": "design-output-v1"
}
```

---

# 93. Agent Versioning

Agent 정의도 Version을 둔다.

```text
design-agent/1.0

implementation-agent/1.2
```

Work Item에는 실제 실행된 Agent Version을 기록한다.

---

# 94. Agent Deployment

Agent는 논리적 컴포넌트다.

반드시 별도 프로세스로 나눌 필요는 없다.

초기에는:

```text
Agent Runtime Service
      ↓
Agent Definition
```

형태로 하나의 Server Application 내에서 실행 가능하다.

---

# 95. Microservice 분리는 후순위

초기부터:

```text
Requirement Service

Design Service

Review Service
```

를 각각 Microservice로 분리할 필요는 없다.

논리적 Agent 경계와 배포 경계를 구분한다.

---

# 96. 권장 초기 배포 구조

```text
onCode Server

├─ API / Session
├─ Orchestrator
├─ Agent Runtime
├─ Context Service
├─ Project Context Service
├─ Tool Gateway
└─ Inference Gateway
```

Agent는 Agent Runtime 내부 논리 Plugin으로 동작한다.

---

# 97. Tool Gateway

Agent Tool Call의 공통 진입점이다.

```text
Agent
  ↓
Tool Gateway
  │
  ├─ Server Tool
  ├─ Local Agent Tool
  └─ MCP Adapter
```

Tool Protocol의 실제 실행 위치를 Agent가 몰라도 된다.

---

# 98. Tool Routing

예:

```text
guide.search
→ Server Guide Service

project.search
→ Server Project Context Service

workspace.read_file
→ Local Agent

dependency.search
→ Nexus Adapter

security.check_dependency
→ Security Service
```

---

# 99. Server Tool Registry

Tool Registry에는 다음 정보를 유지한다.

```text
Tool Name

Provider

Execution Location

Schema

Risk

Timeout

Version
```

---

# 100. Multi-user Isolation

Agent Runtime은 항상 다음 Scope를 유지한다.

```text
user_id

session_id

project_id

workspace_id

work_item_id
```

다른 사용자의 Project Context가 섞이면 안 된다.

---

# 101. Multi-workflow Isolation

같은 사용자라도 Work Item 간 Context를 명시적으로 분리한다.

```text
WI-100
로그인 변경

WI-101
Batch 오류 수정
```

Agent Context Builder가 다른 Work Item 데이터를 임의로 포함하지 않는다.

---

# 102. Project Shared Context

같은 프로젝트의:

```text
Project Profile

Project Index

Development Rules
```

는 여러 Work Item이 공유할 수 있다.

---

# 103. Work Item Private Context

다음은 Work Item별로 분리한다.

```text
Requirement

User Decisions

Design

Change Set

Approval

Review

Build/Test Result
```

---

# 104. Session과 Work Item 관계

사용자가 IDE를 재시작해도 Work Item은 서버에 남아 이어갈 수 있다.

Session은 연결 상태이며 Work Item은 지속적인 작업 상태다.

---

# 105. Agent Resume

Workflow Resume 시 Agent 자체를 복구하지 않는다.

```text
Context Storage
+
Workflow State
```

에서 새 Agent 실행을 시작한다.

즉 Agent Runtime은 stateless하게 유지한다.

---

# 106. 일반 질문 처리

GENERAL_QA의 경우 전체 Agent Pipeline을 실행하지 않는다.

```text
Intent Router
     ↓
General Assistant
```

필요하면 Guide나 Project Search Tool만 사용한다.

---

# 107. 일반 질문과 Senior Developer

Project 관련 일반 질문은 Senior Developer Agent 또는 Code Analysis Agent가 처리할 수 있다.

예:

```text
"이 프로젝트의 인증 구조를 설명해줘."
```

```text
Project Search
→ Code Analysis
→ Answer
```

---

# 108. "찾아줘" 요청

예:

```text
"사용자 권한 검사하는 코드 찾아줘."
```

Workflow:

```text
Intent Classification
→ project.search
→ 필요 Source Read
→ Code Analysis
→ Answer
```

Design/Implementation은 실행하지 않는다.

---

# 109. "검토해줘" 요청

```text
Code Review
+
Guide Check
+
Security Check(optional)
```

로 구성한다.

코드 변경은 하지 않는다.

---

# 110. Agent 수 최소화

초기 MVP에서 Agent 수가 너무 많으면 디버깅이 어렵다.

MVP 통합안:

```text
Senior Developer / Requirement

Guide

Code Analysis / Design

Implementation

Review / Security
```

총 5개 정도로 시작할 수 있다.

---

# 111. 목표 구조로의 확장

안정화 후:

```text
Senior Developer

Requirement

Guide

Code Analysis

Design

Dependency

Implementation

Review

Security
```

로 역할을 세분화한다.

---

# 112. Agent 분리 기준

별도 Agent로 분리할 가치가 있는 조건:

```text
다른 Model이 필요한가

별도 Tool Permission이 필요한가

독립 검증이 중요한가

별도 Prompt/평가 기준이 필요한가

병렬화할 가치가 있는가
```

---

# 113. Review Agent는 반드시 분리

Implementation과 Review는 같은 Agent로 합치지 않는 것을 권장한다.

독립 검증 가치가 크기 때문이다.

---

# 114. Security Agent 분리 시점

초기에는 Review와 통합할 수 있지만 다음 수준이면 분리한다.

```text
SCA

License

Secure Coding Rules

Secret Scan

Vulnerability DB
```

연계가 본격화될 때 독립 Agent가 적절하다.

---

# 115. Senior Developer의 최종 책임

개별 Agent 결과가 충돌하면 Senior Developer가 해결 방향을 결정한다.

예:

```text
Design Agent
→ Library A

Security Agent
→ Library A 사용 금지
```

Senior Developer:

```text
→ Dependency 재검토
```

를 지시한다.

---

# 116. Senior Developer도 Policy를 넘을 수 없음

Senior Developer의 판단보다 다음이 우선한다.

```text
Security Policy

User Approval

Project Policy

Development Guide Mandatory Rule

Local Policy
```

---

# 117. Server Agent Architecture 핵심 흐름

```text
Developer Request
       ↓
Orchestrator
       ↓
Requirement
       ↓
       ├──────── Guide
       │
       └──────── Code Analysis
                     ↓
                   Design
                     ↓
              Developer Approval
                     ↓
                 Dependency
                     ↓
               Implementation
                     ↓
             ┌───────┴────────┐
             ▼                ▼
           Review          Security
             └───────┬────────┘
                     ↓
                Senior Developer
                     ↓
                Change Proposal
                     ↓
                 Local Agent
```

---

# 118. MVP Server 구성

초기 구현 권장 구성:

```text
Orchestrator

Senior Developer Agent

Guide Agent

Code Analysis / Design Agent

Implementation Agent

Review Agent
```

Dependency와 Security는 Service/Tool 중심으로 먼저 구현할 수 있다.

---

# 119. 2차 확장

```text
Requirement Agent 분리

Dependency Agent

Security Agent

Guide Compliance Agent

Parallel Review

Model Routing
```

---

# 120. 핵심 설계 결정

onCode Server Multi-Agent Architecture v1의 핵심 결정은 다음과 같다.

1. Orchestrator와 Agent를 분리한다.
2. Orchestrator는 deterministic Workflow Engine으로 구현한다.
3. Senior Developer Agent가 기술적 의사결정의 중심이 된다.
4. Agent 간 직접 Context 전달을 하지 않는다.
5. Context Storage를 Single Source of Truth로 사용한다.
6. Agent는 가능한 한 Stateless하게 유지한다.
7. Project 탐색은 Project Index에서 시작한다.
8. 실제 구현 전 필요한 Source만 Local Agent에 요청한다.
9. Summary만으로 코드를 구현하지 않는다.
10. Requirement, Design, Implementation, Review 책임을 구분한다.
11. Implementation Agent는 Local Workspace를 직접 수정할 수 없다.
12. Review는 Implementation과 독립적으로 수행한다.
13. Agent Tool Permission을 최소 권한으로 제한한다.
14. Agent 결과는 JSON Schema 기반 Structured Output을 사용한다.
15. Prompt와 Agent 정의를 Versioning한다.
16. LLM Model과 Agent 정의를 분리한다.
17. Inference Gateway를 통해 Model 접근을 통합한다.
18. Agent Retry와 수정 Loop 횟수를 제한한다.
19. Build/Test 실패는 Senior Developer가 원인 유형을 판단한다.
20. Agent private reasoning은 Context Storage에 저장하지 않는다.
21. Source/문서의 자연어를 시스템 지시사항으로 취급하지 않는다.
22. MVP에서는 Agent 수를 줄이고 안정화 후 세분화한다.
23. Agent는 논리적 경계이며 초기부터 Microservice로 분리하지 않는다.
24. Tool Gateway가 Server/Local/MCP Tool을 통합 Routing한다.
25. Multi-user, Multi-project, Multi-work-item Context를 엄격히 격리한다.