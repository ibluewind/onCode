# SPEC-15 onCode MVP Scope / Implementation Roadmap v1

## 1. 목적

본 명세는 onCode의 MVP 범위와 실제 구현 순서를 정의한다.

지금까지의 SPEC-01~14는 전체 Architecture와 Subsystem을 정의하였다.

본 문서는 이를 다음 관점으로 재구성한다.

```text
무엇을 먼저 만들 것인가

어떤 기능을 MVP에 포함할 것인가

어떤 기능을 후순위로 미룰 것인가

Server / Local Agent / IDE를 어떤 순서로 구현할 것인가

언제 각 Subsystem을 통합할 것인가

어떤 기준으로 MVP 완료를 판단할 것인가
```

---

# 2. MVP의 목표

MVP의 핵심 목표는 다음 하나의 End-to-End Flow를 안정적으로 완성하는 것이다.

```text
Developer Request
    ↓
Server Analysis
    ↓
Project Context 조회
    ↓
Design 생성
    ↓
Developer Design Approval
    ↓
Implementation Change 생성
    ↓
Local Diff 생성
    ↓
Developer Code Approval
    ↓
Local Apply
    ↓
Build
    ↓
Test
    ↓
Result
```

이 Flow가 안정적으로 동작하기 전에는 고급 Multi-Agent 기능이나 복잡한 Search 기능을 우선하지 않는다.

---

# 3. MVP 핵심 원칙

MVP는 다음 원칙으로 범위를 제한한다.

```text
End-to-End 먼저

정확성 > 기능 수

안전한 Apply > 자동화 정도

구조화된 Protocol > UI 고급 기능

재현 가능한 상태 > Agent 수 증가

기본 검색 > 고급 검색

실제 Build/Test 성공 > Agent 설명 품질
```

---

# 4. MVP에서 반드시 증명해야 하는 것

MVP는 최소 다음을 기술적으로 증명해야 한다.

1. Server와 Local Agent가 안정적으로 연결된다.
2. Server가 Local Workspace를 직접 접근하지 않는다.
3. 필요한 Source만 Local Agent에 요청할 수 있다.
4. Server가 구현 설계를 만든다.
5. 개발자가 설계를 승인/반려할 수 있다.
6. Server가 Proposed Change를 생성한다.
7. Local Agent가 현재 Workspace 기준 Diff를 생성한다.
8. 개발자가 실제 Diff를 승인할 수 있다.
9. Local Agent가 승인된 Change만 적용한다.
10. 변경 이후 Build/Test를 수행한다.
11. 실패하면 구조화된 Error가 Server로 돌아간다.
12. Server가 수정 Loop를 수행할 수 있다.
13. 모든 주요 상태가 Context/Workflow Storage에 남는다.
14. Workflow가 중간에 끊겨도 Resume 가능하다.
15. 위험 작업이 Approval/Policy를 우회할 수 없다.

---

# 5. MVP 사용자 시나리오

대표 시나리오:

```text
Developer:
"로그인 실패가 5회 이상이면 계정을 잠가줘."
```

onCode:

```text
1. 관련 코드 탐색

2. 필요한 Source 요청

3. 요구사항 분석

4. 구현 설계 제시

5. 사용자 승인

6. 코드 생성

7. Diff 표시

8. 사용자 승인

9. 파일 적용

10. Build

11. Test

12. 완료 결과 표시
```

---

# 6. MVP Intent 범위

초기 지원 Intent:

```text
GENERAL_QA

PROJECT_SEARCH

CODE_EXPLAIN

IMPLEMENT

MODIFY

FIX

CODE_REVIEW

BUILD

TEST

GIT_STATUS
```

후순위:

```text
REFACTOR_LARGE_SCOPE

GIT_REBASE

GIT_PUSH

DEPENDENCY_MIGRATION

MULTI_PROJECT_CHANGE
```

---

# 7. MVP 전체 구성

```text
IDE Extension
     ↓
Local Agent
     ↓
onCode Server
     │
     ├─ Workflow Orchestrator
     ├─ Agent Runtime
     ├─ Context Service
     ├─ Project Context Service
     ├─ Tool Gateway
     ├─ Guide Retrieval
     └─ Inference Gateway
```

외부/기존 시스템:

```text
PostgreSQL

Internal LLM

Nexus

Internal Git

Authentication

Guide Repository
```

---

# 8. 구현 Phase 개요

권장 구현 순서:

```text
Phase 0
Foundation

Phase 1
Local Agent Core

Phase 2
Server Workflow Core

Phase 3
IDE Interaction

Phase 4
Project Intelligence

Phase 5
Implementation Loop

Phase 6
Guide Integration

Phase 7
Dependency / Security

Phase 8
Operations / Hardening
```

---

# 9. Phase 0 — Foundation

목표:

```text
공통 Protocol과 Repository 구조 확정
```

구현:

```text
SPEC-01 Tool Protocol

공통 DTO

Error Code

ID Generation

Protocol Version

JSON Schema

Shared Test Fixtures
```

---

# 10. Phase 0 산출물

```text
protocol/

schemas/

common-model/

error-codes/

test-fixtures/
```

예:

```text
tool-request.schema.json

tool-response.schema.json

workflow-event.schema.json

approval.schema.json
```

---

# 11. Phase 0 중요 결정

초기부터 다음을 확정한다.

```text
message_id

call_id

session_id

project_id

workspace_id

work_item_id

workflow_id

approval_id

change_set_id
```

이 ID 구조를 나중에 변경하면 전체 시스템 수정 비용이 크다.

---

# 12. Phase 1 — Local Agent Core

가장 먼저 실제 Local Agent를 구현한다.

이유:

onCode의 가장 중요한 차별점은 Server가 파일을 직접 수정하지 않고 Local Agent를 통해 안전하게 작업하는 구조이기 때문이다.

---

# 13. Phase 1 기능

필수:

```text
Server Connector

Tool Registry

Local Agent Core

Workspace Root 관리

Policy Engine 기본

File Read

File Search

Diff

Apply

Git Status

Build Run

Test Run

Heartbeat
```

---

# 14. Phase 1 Tool

최소 Tool:

```text
system.ping

system.get_capabilities

workspace.list

workspace.read_file

workspace.read_files

workspace.search

workspace.propose_changes

workspace.apply_changes

git.status

git.diff

build.run

test.run
```

---

# 15. Phase 1 보안

반드시 포함:

```text
Workspace Boundary

Canonical Path 검증

Symlink Escape 차단

File Hash

Approval ID 검증

Secret 기본 Masking

Shell 기본 비활성
```

---

# 16. Phase 1 테스트

독립적으로 Local Agent Tool을 호출해 다음을 검증한다.

```text
파일 읽기

존재하지 않는 파일

Workspace 밖 요청

Diff 생성

Hash mismatch

Apply 성공

Apply 실패

Build 성공/실패

Test 성공/실패
```

---

# 17. Phase 1 완료 기준

Server 없이도 Test Client를 통해:

```text
Tool Request
→ Local Agent
→ Tool Result
```

가 안정적으로 동작하면 완료.

---

# 18. Phase 2 — Server Workflow Core

Local Agent 이후 Server의 핵심 Workflow를 구현한다.

---

# 19. Phase 2 구성

```text
API

Session 기본

Workflow Orchestrator

Context Service

Tool Gateway

Agent Runtime

Inference Gateway

PostgreSQL
```

---

# 20. Phase 2 Agent 최소 구성

초기 Agent 수를 줄인다.

권장:

```text
Senior Developer

Code Analysis / Design

Implementation

Review
```

Requirement는 Senior Developer에 포함 가능.

Security/Dependency는 후순위.

---

# 21. Phase 2 Workflow

최초 구현 Workflow:

```text
RECEIVED
 ↓
CLASSIFYING
 ↓
DISCOVERING_CONTEXT
 ↓
DESIGNING
 ↓
WAITING_DESIGN_APPROVAL
 ↓
IMPLEMENTING
 ↓
REVIEWING
 ↓
PREPARING_CHANGE
 ↓
WAITING_CODE_APPROVAL
 ↓
APPLYING_CHANGE
 ↓
BUILDING
 ↓
TESTING
 ↓
COMPLETED
```

---

# 22. Phase 2에서 제외 가능한 State

초기에는 다음을 단순화할 수 있다.

```text
SECURITY_REVIEWING

ANALYZING_DEPENDENCY

PARTIAL_APPROVAL

WAITING_RESOURCE
```

---

# 23. Phase 2 Context Entity

최소:

```text
WORK_ITEM

WORKFLOW

WORKFLOW_TRANSITION

REQUIREMENT

DESIGN

APPROVAL

CHANGE_SET

BUILD_RESULT

TEST_RESULT

TOOL_CALL
```

---

# 24. Phase 2 Context Tool

```text
context.get

context.put
```

부터 구현.

고급 `context.query`는 후순위.

---

# 25. Phase 2 Agent Runtime

공통 구조:

```text
Agent Definition

Prompt Template

Context Builder

Model Client

Output Schema Validator

Result Writer
```

---

# 26. Phase 2 LLM 연결

Inference Gateway를 통해 내부 Model 호출.

처음에는 모델 1개로도 가능하다.

```text
reasoning/coding unified model
```

후속으로 분리.

---

# 27. Phase 2 완료 기준

Mock Project Context를 사용해:

```text
User Request
→ Design
→ Approval
→ Implementation
```

까지 Server 내부 Workflow가 동작하면 완료.

---

# 28. Phase 3 — IDE Interaction

이제 IDE에서 실제 사용자 Interaction을 연결한다.

첫 번째 IDE는 하나만 선택하는 것을 권장한다.

---

# 29. 첫 IDE 권장

권장:

```text
VS Code
```

이유:

```text
Extension 개발 난이도

Webview

Diff API

배포 편의
```

단 실제 조직 표준 IDE가 IntelliJ/Eclipse라면 우선순위를 바꾼다.

---

# 30. Phase 3 기능

```text
Chat

Current File Context

Selection Context

Progress

Question

Design Review

Design Approval

Diff Review

Code Approval

Build/Test Result

Error Notification
```

---

# 31. IDE ↔ Local Agent

초기:

```text
localhost WebSocket + JSON
```

정도로 구현 가능.

---

# 32. Phase 3에서 제외

```text
Inline Completion

Code Lens

Hunk Approval

Advanced History

Multi-root UI

Architecture Visualization
```

---

# 33. Phase 3 완료 기준

실제 IDE에서:

```text
사용자 요청
→ Server Design
→ IDE 승인
```

까지 완료.

---

# 34. Phase 4 — Project Intelligence

이제 실제 프로젝트를 Server가 효율적으로 이해하도록 한다.

---

# 35. Phase 4 MVP

```text
Project Detector

Workspace Scanner

File Hash

Language Detection

Basic Parser

File Summary

Symbol Extraction

PROJECT_INDEX

Delta Update
```

---

# 36. 초기 언어 우선순위

사용자 요구 기준:

```text
1. Java
2. JavaScript / TypeScript
3. Python
```

Framework:

```text
Spring / Spring Boot

React / Vue

Node.js

eGovFramework
```

---

# 37. Java 우선 이유

onCode 주요 기업/폐쇄망 대상에서:

```text
Spring

eGovFramework
```

지원 가치가 높으므로 Java Parser를 먼저 안정화하는 것이 좋다.

---

# 38. Phase 4 Project Index 최소 Schema

```text
Project

Module

File

Symbol

Dependency

Revision

Hash
```

Relation은 Basic Import 정도부터 시작.

---

# 39. Phase 4 Summary

LLM Summary는 초기 필수는 아니다.

먼저:

```text
File Name

Package

Class

Methods

Javadoc
```

기반 heuristic summary로 시작 가능.

---

# 40. LLM Summary 도입

구조 Index가 안정화된 후:

```text
File Summary

Important Method Summary
```

를 추가.

---

# 41. Incremental Index

초기부터 File Watcher를 구현한다.

전체 프로젝트를 매번 Scan하는 구조로 시작하지 않는 것이 좋다.

---

# 42. Phase 4 완료 기준

Server가:

```text
"로그인 처리"
```

검색 시:

```text
LoginController

AuthService

UserRepository
```

등 관련 파일 후보를 찾을 수 있어야 한다.

---

# 43. Phase 5 — End-to-End Implementation Loop

이 단계에서 첫 번째 진짜 MVP가 완성된다.

---

# 44. E2E 흐름

```text
Developer
 ↓
IDE
 ↓
Server
 ↓
project.search
 ↓
workspace.read_files
 ↓
Design
 ↓
Approval
 ↓
Implementation
 ↓
workspace.propose_changes
 ↓
Local Diff
 ↓
Approval
 ↓
workspace.apply_changes
 ↓
Build
 ↓
Test
```

---

# 45. Fix Loop

Build 실패:

```text
Build Result
 ↓
Senior Developer
 ↓
Code Error 판단
 ↓
Relevant Source 추가 Read
 ↓
Implementation Fix
 ↓
Diff
 ↓
Approval
 ↓
Apply
 ↓
Build
```

---

# 46. Fix Loop 제한

초기:

```text
Build Fix max 3

Test Fix max 3

Implementation Review max 5
```

초과 시 사용자에게 상태를 반환.

---

# 47. MVP Build Tool

최초에는 하나의 기술 Stack부터 지원해도 된다.

권장 첫 Stack:

```text
Java + Maven + Spring Boot
```

---

# 48. 두 번째 Stack

다음:

```text
Node/npm + React/Vue
```

---

# 49. 세 번째 Stack

```text
Python + pytest
```

---

# 50. MVP Review Agent

Review 기준:

```text
Compile 가능성

Design Compliance

Existing Code Pattern

Obvious Error

Naming

Null/Error Handling
```

Security 상세 검사는 후속.

---

# 51. Phase 5 완료 기준

실제 Sample Project에서 다음 시나리오가 성공해야 한다.

```text
기능 추가

기존 메서드 수정

Build Error 수정

Test Failure 수정
```

---

# 52. MVP 공식 완료 시점

실질적인 MVP는 Phase 5 완료 시점으로 보는 것이 좋다.

이 상태에서 이미:

```text
Cursor-like request

Design approval

Diff approval

Local apply

Build/test fix
```

의 핵심 가치가 구현된다.

---

# 53. Phase 6 — Guide Integration

기본 코딩 Loop가 안정화된 후 Guide를 연결한다.

이 순서가 중요하다.

Guide Search를 먼저 정교하게 만들고 Core Loop가 불안정하면 전체 개발 속도가 늦어진다.

---

# 54. Phase 6-A Guide Ingestion

SPEC-09 구현.

최초 지원:

```text
PDF

DOCX

PPTX

HWP/HWPX

Markdown
```

---

# 55. Phase 6-A Pipeline

```text
Upload
 ↓
Conversion
 ↓
Markdown
 ↓
Section Split
 ↓
Summary
 ↓
Keyword
 ↓
Admin Review
 ↓
Publish
```

---

# 56. Phase 6-A Console

최소:

```text
Upload

Status

Markdown Preview

Section Edit

Category

Summary

Keyword

Publish
```

---

# 57. Phase 6-B Runtime Retrieval

SPEC-10 구현.

최초 검색:

```text
Metadata Filter

Keyword

Title

Summary FTS

Category

Technology

Priority
```

---

# 58. Vector DB 제외

MVP에서는 사용하지 않는다.

PostgreSQL 기반으로 충분히 시작한다.

---

# 59. Mandatory Rule

Phase 6에서 반드시 넣을 가치가 있다.

```text
PROJECT_RULE

ORGANIZATION_RULE

REQUIRED

SECURITY_REQUIRED
```

이유는 Guide Search 기능의 핵심 가치가 단순 정보 검색보다 “규칙 적용”에 있기 때문이다.

---

# 60. Guide Integration Workflow

```text
Requirement
 ↓
Guide Search
 ↓
Design
 ↓
Guide Reference
 ↓
Implementation
 ↓
Review against same Guide
```

---

# 61. Phase 6 완료 기준

예:

```text
프로젝트 가이드:
Controller에서 Repository 직접 호출 금지
```

가 있을 때 Agent가:

```text
Controller → Service → Repository
```

방식으로 설계하고 Review가 이를 검증할 수 있어야 한다.

---

# 62. Phase 7 — Dependency / Security

Guide Integration 이후 Nexus와 Security를 연결한다.

---

# 63. Dependency MVP

SPEC-11 중 초기:

```text
Current Dependency 조회

Nexus Search

Nexus Resolve

Version Selection

Dependency Decision
```

---

# 64. Security MVP

```text
Offline Vulnerability Check

Critical/High Block

License Basic Check

Public Registry Block
```

---

# 65. 신규 Dependency Workflow

```text
Design
 ↓
Dependency Required
 ↓
Current Dependency
 ↓
Nexus
 ↓
Security
 ↓
Decision
 ↓
Design Approval
```

---

# 66. Security Agent

이 단계에서 Review와 Security Agent를 분리할 수 있다.

---

# 67. Dependency Post-check

실제 Resolve 이후:

```text
Actual Dependency Graph

→ Security Recheck
```

추가.

---

# 68. Phase 7 완료 기준

취약 Dependency를 요청하는 Test Case에서:

```text
자동 선택 금지
→ Safe Version
```

또는:

```text
BLOCK
```

되어야 한다.

---

# 69. Phase 8 — Authentication / Operations / Hardening

Pilot/Production 전 안정화 단계.

---

# 70. Authentication

SPEC-12 구현:

```text
Internal IdP

Session

Local Agent Registration

Project RBAC

Workspace Scope
```

---

# 71. Observability

SPEC-13:

```text
Audit

Structured Log

Metrics

Workflow Timeline

Local Agent Health
```

---

# 72. Deployment

SPEC-14:

```text
2+ Runtime Server

PostgreSQL Backup/HA

Shared Artifact Storage

Inference Gateway

Health Check
```

---

# 73. MVP와 Production Ready 구분

중요하다.

## MVP

```text
기능 동작 검증
```

## Pilot

```text
실제 개발자 일부 사용
```

## Production

```text
HA / Audit / Security / 운영 체계
```

로 분리한다.

---

# 74. Stage 정의

권장 단계:

```text
P0 Prototype

P1 Technical MVP

P2 Developer Pilot

P3 Security Pilot

P4 Production
```

---

# 75. P0 Prototype

목표:

```text
Server ↔ Local Agent Tool Calling 검증
```

기능:

```text
read_file

apply_changes

build
```

CLI 기반으로도 가능.

IDE 없음.

---

# 76. P1 Technical MVP

목표:

```text
완전한 E2E 개발 Flow
```

포함:

```text
VS Code

Design Approval

Diff Approval

Java Project Index

Maven Build/Test

Basic Review
```

---

# 77. P2 Developer Pilot

추가:

```text
Guide

Multiple Projects

Git Commit

Node/Python

Workflow Resume

Basic Auth
```

---

# 78. P3 Security Pilot

추가:

```text
Nexus

Vulnerability

License

Security Agent

Audit

Policy

RBAC
```

---

# 79. P4 Production

추가:

```text
HA

Backup/Restore

Monitoring

Alert

Deployment Automation

Upgrade/Rollback
```

---

# 80. 구현 Workstream

병렬 개발을 위해 다음 Workstream으로 나눈다.

```text
WS-A Protocol / Core

WS-B Local Agent

WS-C Server / Workflow

WS-D Agent Runtime

WS-E IDE Extension

WS-F Project Intelligence

WS-G Guide

WS-H Dependency / Security

WS-I Platform / Operations
```

---

# 81. WS-A Protocol

담당:

```text
SPEC-01

DTO

Schemas

Error

Version

Compatibility
```

가장 먼저 시작.

---

# 82. WS-B Local Agent

담당:

```text
SPEC-04

SPEC-05 일부

SPEC-08 Local Policy

SPEC-12 Local Session
```

---

# 83. WS-C Server

담당:

```text
SPEC-02

SPEC-03

Tool Gateway

Context

Workflow
```

---

# 84. WS-D Agent Runtime

담당:

```text
SPEC-06

Prompt

Context Builder

Inference Gateway
```

---

# 85. WS-E IDE

담당:

```text
SPEC-07
```

Local Agent API가 확정된 후 본격 구현.

---

# 86. WS-F Project Intelligence

담당:

```text
SPEC-05

Parser

Indexer

Search
```

Local Agent Team과 밀접하게 협업.

---

# 87. WS-G Guide

담당:

```text
SPEC-09

SPEC-10
```

Core MVP와 독립적으로 병렬 진행 가능.

---

# 88. WS-H Dependency/Security

담당:

```text
SPEC-11

SPEC-08 일부
```

기존 Nexus/검색 시스템과 연계.

---

# 89. WS-I Platform

담당:

```text
SPEC-12

SPEC-13

SPEC-14
```

Pilot 시점부터 강화.

---

# 90. 구현 의존성

대략적인 Dependency:

```text
SPEC-01
 ↓
Local Agent
 ↓
Tool Gateway
 ↓
Workflow
 ↓
Agent Runtime
 ↓
IDE
```

Project Intelligence는 Local Agent Core 이후.

---

# 91. Critical Path

실제 Critical Path:

```text
Protocol

→ Local File Tool

→ Server Tool Gateway

→ Workflow

→ Agent Runtime

→ IDE Approval

→ Apply

→ Build/Test
```

이 경로를 먼저 닫는다.

---

# 92. Guide는 Critical Path가 아님

초기 MVP에서는 Guide 없이도 End-to-End를 완성할 수 있어야 한다.

이후 Guide를 넣어서 품질을 높인다.

---

# 93. Nexus도 초기 Critical Path가 아님

기존 Dependency만 사용하는 Feature로 MVP를 검증하면 된다.

신규 Dependency 처리는 Phase 7.

---

# 94. Multi-Agent도 최소화

초기부터 9개의 Agent를 전부 구현하지 않는다.

---

# 95. Agent 단계적 확장

P1:

```text
Senior Developer

Implementation

Review
```

P2:

```text
Guide

Code Analysis / Design 분리
```

P3:

```text
Requirement

Dependency

Security
```

---

# 96. Senior Developer MVP

초기에는 다음을 포함할 수 있다.

```text
Intent

Requirement

Context Need

Design
```

안정화 후 역할 분리.

---

# 97. 고급 Workflow 후순위

다음은 Production 이후에도 확장 가능하다.

```text
Parallel Agent Debate

Multiple Solution Candidates

Automatic Architecture Refactoring

Cross-repository implementation

Multi-agent planning tree
```

---

# 98. MVP Tool 우선순위

P0:

```text
workspace.read_file

workspace.apply_changes

build.run
```

P1:

```text
workspace.read_files

workspace.search

workspace.propose_changes

test.run

git.status

git.diff
```

P2:

```text
project.search

guide.*

git.commit
```

P3:

```text
dependency.*

security.*
```

---

# 99. Shell Tool

`shell.execute`는 MVP 필수 기능으로 만들지 않는 것을 권장한다.

이유:

```text
보안

Policy Complexity

Command Parsing

Audit
```

전용 Tool로 대부분의 개발 Flow를 검증할 수 있다.

---

# 100. Git MVP

초기:

```text
status

diff
```

P2:

```text
commit
```

후순위:

```text
push

merge

rebase

reset

clean
```

---

# 101. IDE MVP UX

최소 UX:

```text
Chat

Progress

Design Card

Approve / Reject

Diff

Approve / Reject

Result
```

이 여섯 가지가 핵심이다.

---

# 102. IDE 고급 UX 후순위

```text
Inline completion

Ghost text

Multi-file inline edit

Hunk acceptance

Composer-style UI

Architecture diagram
```

---

# 103. Project Intelligence 정확도 우선순위

초기에는:

```text
File discovery accuracy
>
Symbol relationship accuracy
>
Summary quality
>
Call graph completeness
```

로 우선순위를 둔다.

---

# 104. Parser 기반 우선

LLM이 Project Tree를 분석하는 방식으로 MVP를 만들지 않는다.

```text
Parser / Scanner
→ structural truth
```

를 먼저 구현한다.

---

# 105. Project Search 평가

Golden Query를 만든다.

예:

```text
"로그인"
Expected:
LoginController
AuthService

"사용자 저장"
Expected:
UserRepository
```

---

# 106. Project Search KPI

초기 예:

```text
Expected relevant file in Top-5
>= 90%
```

정도로 목표를 둘 수 있다.

---

# 107. Agent Output Schema

모든 Agent는 초기부터 Structured Output을 사용한다.

나중에 JSON Schema를 붙이는 방식은 피한다.

---

# 108. Prompt Test

Agent별 Prompt에 regression test case를 둔다.

예:

```text
Requirement extraction

Need more context

Design generation

Build error classification
```

---

# 109. Sample Projects

통합 테스트용 프로젝트를 별도 유지한다.

예:

```text
sample-java-spring

sample-egov

sample-react

sample-vue

sample-python
```

---

# 110. Sample Java 프로젝트

최초 가장 중요.

포함:

```text
Controller

Service

Repository

JUnit

Maven

Intentional bugs
```

---

# 111. Golden Workflow

CI에서 반복 테스트할 대표 Workflow를 만든다.

예:

```text
"Greeter에 bye 메서드를 추가해줘."
```

Expected:

```text
Design

Diff

Build Success

Test Success
```

---

# 112. 테스트 계층

```text
Unit

Component

Contract

Integration

E2E
```

모두 필요하지만 E2E를 일찍 만든다.

---

# 113. Protocol Contract Test

Server와 Local Agent는 서로 다른 Repository/언어로 구현될 수 있으므로 JSON Schema 기반 Contract Test가 중요하다.

---

# 114. Tool Integration Test

실제 Temp Workspace에서:

```text
Read

Diff

Apply

Build

Test
```

검증.

---

# 115. Workflow State Test

모든 주요 Transition을 테스트한다.

예:

```text
Design reject

Code reject

Build fail

Workspace stale

User cancel
```

---

# 116. Approval Test

반드시 테스트:

```text
Approval 재사용

Expired approval

Wrong change set

Changed hash

Concurrent approval
```

---

# 117. Stale Workspace Test

핵심 시나리오:

```text
Server design
 ↓
Proposed change
 ↓
Developer manually edits file
 ↓
Apply attempt
 ↓
WORKSPACE_FILE_CHANGED
```

반드시 안전하게 실패해야 한다.

---

# 118. Server Crash Test

```text
WAITING_DESIGN_APPROVAL
```

상태에서 Server 재시작 후 Resume 검증.

---

# 119. Local Disconnect Test

Build 직전 Local Agent 연결 해제 → 재접속 → Revision 확인 → Resume.

---

# 120. Security Test

MVP에서도 최소:

```text
../ file access

absolute external path

invalid approval

public URL command
```

등을 차단해야 한다.

---

# 121. Performance MVP

초기 성능 목표는 지나치게 공격적으로 두지 않는다.

관측 우선.

측정:

```text
project.search

file read

LLM response

diff

build
```

---

# 122. E2E Latency

LLM/Build 특성상 절대시간보다 Breakdown을 먼저 측정한다.

```text
Server reasoning

Model queue

Local IO

Build

Human wait
```

---

# 123. Observability 조기 도입

SPEC-13 전체는 후순위라도 다음은 P1부터 반드시 넣는다.

```text
trace_id

work_item_id

tool_call_id

structured error

basic audit
```

없으면 통합 디버깅이 매우 어렵다.

---

# 124. DB Migration

Phase 2부터 Migration Tool을 사용한다.

Schema를 수동으로 운영하지 않는다.

---

# 125. Version 관리

P0부터:

```text
protocol version

DB schema version

agent version

local agent version
```

을 기록.

---

# 126. Feature Flag

불완전한 기능:

```text
Guide

Security Agent

Dependency

Git Commit
```

등을 Feature Flag로 켜고 끌 수 있게 한다.

---

# 127. 개발 Repository 권장 구조

하나의 Monorepo 또는 명확한 Multi-repo 전략 필요.

예:

```text
oncode/

├─ server/
├─ local-agent/
├─ ide/
│  └─ vscode/
├─ protocol/
├─ guide-ingestion/
├─ guide-admin/
├─ deployment/
└─ samples/
```

---

# 128. Monorepo 장점

초기에는 Protocol/DTO 변경이 많으므로 Monorepo가 편리할 수 있다.

하지만 Server와 IDE 기술 Stack이 크게 다르면 Multi-repo도 가능.

중요한 것은 Protocol Schema를 중앙 관리하는 것이다.

---

# 129. API First

새 Tool/Message 개발 순서:

```text
Schema

Contract Test

Provider

Consumer
```

순으로 한다.

---

# 130. Local Agent 언어 선택 기준

필요 조건:

```text
Cross-platform

Process Control

Filesystem

Git

Parser Integration

IDE IPC

Long-running daemon
```

언어 선택은 별도 기술 결정 문서에서 확정한다.

---

# 131. Server 기술 선택 기준

필요:

```text
Concurrent Workflow

WebSocket

DB Transaction

LLM Integration

Job Scheduling

Security

Observability
```

기존 조직 기술 Stack과 운영 역량을 우선한다.

---

# 132. 기술 선택과 Architecture 분리

SPEC은 Java/Spring 등 특정 Server 구현 기술에 불필요하게 고정하지 않는다.

---

# 133. Definition of Done — Tool

Tool 완료 기준:

```text
Schema

Implementation

Error Mapping

Policy

Audit

Unit Test

Contract Test

Documentation
```

---

# 134. Definition of Done — Workflow State

```text
Entry Condition

Action

Exit Condition

Error Transition

Persistence

Resume

Test
```

가 있어야 완료.

---

# 135. Definition of Done — Agent

```text
Prompt Version

Allowed Tools

Input Context

Output Schema

Validation

Fallback/Error

Evaluation Test
```

완료.

---

# 136. Definition of Done — IDE Feature

```text
Protocol

UI

Error handling

Reconnect

Cancellation

Accessibility/basic UX
```

---

# 137. Definition of Done — Project Parser

```text
Valid source

Invalid source

Large file

Generated file

Incremental change

Golden sample
```

테스트.

---

# 138. Technical Debt 관리

MVP 단순화 항목을 명확히 기록한다.

예:

```text
Single model

Single IDE

PostgreSQL-only session

No message broker

Basic FTS

Basic Java parser
```

후속 Roadmap과 연결한다.

---

# 139. 구현 우선순위 판단 기준

기능 추가 전 질문:

```text
E2E 개발 Flow에 필요한가?

안전성을 높이는가?

디버깅 가능성을 높이는가?

실제 개발자 가치가 큰가?
```

아니면 후순위.

---

# 140. 피해야 할 초기 개발 방향

다음은 MVP 지연 위험이 크다.

```text
Agent 수부터 많이 만들기

Vector DB 먼저 구축

전체 Call Graph 완성

세 IDE 동시 개발

복잡한 Git 자동화

완벽한 Microservice

고급 UI부터 개발
```

---

# 141. 먼저 해야 할 방향

```text
Tool Protocol

Local Agent

Workflow

Approval

Diff

Build/Test

실제 프로젝트 검색
```

이다.

---

# 142. 첫 번째 Vertical Slice

가장 먼저 구현할 Vertical Slice:

```text
User:
"Greeter.java의 hello 반환값을 수정해줘."

Server:
read_file

Design

Approval

Generate full file

Local Diff

Approval

Apply

Maven test
```

---

# 143. 두 번째 Vertical Slice

```text
"새 메서드 추가해줘."
```

Multi-line edit 검증.

---

# 144. 세 번째 Vertical Slice

```text
"Service 기능 추가해줘."
```

Multi-file edit 검증.

---

# 145. 네 번째 Vertical Slice

의도적 Compile Error를 만들어 Fix Loop 검증.

---

# 146. 다섯 번째 Vertical Slice

개발자가 Approval 전에 파일을 직접 수정해서 Stale Workspace 검증.

---

# 147. Pilot 대상 Feature

실제 개발자 Pilot에는 다음 정도가 있으면 가치가 있다.

```text
Project Q&A

Code Explain

Small Modify

Bug Fix

Test Generate

Build Fix

Code Review
```

---

# 148. Pilot에서 제한할 기능

```text
대형 Refactor

100+ File Change

DB Migration 자동 실행

Git Push

Destructive Command
```

---

# 149. Pilot Risk Limit

Change Set 제한 예:

```text
max files = 20

max changed lines = configurable
```

초기 안전장치로 유용하다.

---

# 150. 사용자 Feedback

Pilot에서 수집할 항목:

```text
Design 적절성

Diff 품질

불필요한 질문 수

검색 정확도

Build Fix 성공률

승인 피로도

응답 속도
```

---

# 151. 품질 KPI

초기 후보:

```text
Build Success after generation

Test Success

Review Pass

Design Approval First-pass Rate

Average Fix Loops

Project Search Top-5 Accuracy

Stale Apply Prevention
```

---

# 152. Safety KPI

```text
Unauthorized Apply = 0

Outside Workspace Access = 0

Approval Bypass = 0

Public Registry Access = 0
```

이들은 성공률이 아니라 반드시 0이어야 하는 항목이다.

---

# 153. Reliability KPI

```text
Workflow Resume Success

Local Agent Reconnect

Duplicate Tool Execution

Lost Approval
```

측정.

---

# 154. MVP Exit Criteria

P1 Technical MVP 완료 조건 예:

```text
1. Java/Maven Sample Project 지원

2. VS Code 지원

3. Project Search 가능

4. Design Approval 가능

5. Multi-file Diff 가능

6. Code Approval 가능

7. Atomic Apply 가능

8. Build/Test 가능

9. Build Fix Loop 가능

10. Stale Workspace 차단

11. Basic Audit

12. Server Restart Resume
```

---

# 155. Pilot Exit Criteria

P2:

```text
실제 내부 프로젝트 3개 이상

개발자 5~10명

핵심 Workflow 안정성 확인

Guide Retrieval 적용

Node 또는 Python 추가

Git Commit 지원
```

정도.

---

# 156. Security Pilot Exit Criteria

P3:

```text
Nexus-only

Offline Vulnerability

Security Agent

RBAC

Audit Search

Policy Enforcement

Secret Filtering
```

---

# 157. Production Exit Criteria

P4:

```text
HA

Backup/Restore

Monitoring/Alert

Rolling Upgrade

Agent Version Control

Security Review 완료

운영 Runbook
```

---

# 158. 추천 개발 순서 요약

```text
1. Protocol

2. Local Agent File Tool

3. Server Tool Gateway

4. Workflow / Context

5. Agent Runtime

6. IDE Design Approval

7. Diff / Code Approval

8. Apply

9. Build/Test

10. Project Intelligence

11. Fix Loop

12. Guide Ingestion

13. Guide Retrieval

14. Dependency/Nexus

15. Security

16. Auth/RBAC

17. Observability

18. HA/Production
```

---

# 159. SPEC별 구현 매핑

```text
SPEC-01
→ 가장 먼저

SPEC-02
→ Phase 2

SPEC-03
→ Phase 2

SPEC-04
→ Phase 1

SPEC-05
→ Phase 4

SPEC-06
→ Phase 2~5

SPEC-07
→ Phase 3

SPEC-08
→ Phase 1부터 부분 적용, Phase 7 강화

SPEC-09
→ Phase 6

SPEC-10
→ Phase 6

SPEC-11
→ Phase 7

SPEC-12
→ Phase 8 / Pilot 전

SPEC-13
→ P1부터 최소 기능, Phase 8 강화

SPEC-14
→ Architecture는 초기 반영, Production에서 완성
```

---

# 160. 코드 착수 전 추가 문서

15개 SPEC 이후 별도 Architecture SPEC를 더 늘리기보다 실제 구현에 필요한 다음 문서를 만드는 편이 좋다.

```text
ADR — Architecture Decision Record

API / JSON Schema

DB DDL

Sequence Diagram

Module Interface

Test Scenario

Backlog / Epic
```

---

# 161. 우선 작성할 ADR 예

```text
ADR-001 Server 구현 언어/Framework

ADR-002 Local Agent 구현 언어

ADR-003 Server ↔ Local Agent Transport

ADR-004 IDE ↔ Local Agent IPC

ADR-005 PostgreSQL / Session Store

ADR-006 Parser Library 선택

ADR-007 Inference Engine

ADR-008 Artifact Storage
```

---

# 162. SPEC에서 구현으로 넘어가는 방식

권장:

```text
SPEC
 ↓
ADR
 ↓
Interface / Schema
 ↓
Epic
 ↓
Story / Task
 ↓
Code
```

---

# 163. 첫 개발 Epic

예:

```text
EPIC-001
Local Agent Workspace Tools
```

Story:

```text
workspace.read_file

workspace.read_files

workspace.search

workspace.propose_changes

workspace.apply_changes
```

---

# 164. 두 번째 Epic

```text
EPIC-002
Tool Protocol / Server Gateway
```

---

# 165. 세 번째 Epic

```text
EPIC-003
Implementation Workflow
```

---

# 166. 네 번째 Epic

```text
EPIC-004
VS Code Developer Experience
```

---

# 167. 다섯 번째 Epic

```text
EPIC-005
Project Intelligence
```

---

# 168. 개발팀 병렬화 예

초기 3개 팀이면:

```text
Team A
Server / Workflow

Team B
Local Agent / Project Intelligence

Team C
IDE / Integration
```

Guide/Dependency는 Core MVP 이후 별도 병렬화.

---

# 169. 핵심 위험 요소

초기 위험:

```text
Agent Context 과다

Project Search 부정확

Local Diff/Apply Race

Persistent Connection 불안정

Workflow Resume

LLM Structured Output 실패

Build Result 구조화
```

---

# 170. 위험 대응 우선순위

특히 다음 세 개는 초기에 집중 테스트한다.

```text
1. File Hash / Stale Workspace

2. Workflow Resume

3. Structured Tool / Agent Output
```

---

# 171. 성공 가능성을 높이는 전략

onCode를 처음부터 Cursor 전체 기능을 복제하는 프로젝트로 보지 않는다.

첫 목표는:

```text
"내부 폐쇄망에서
안전하게 설계 승인 후 코드를 수정하고
실제 로컬 Build/Test까지 검증하는
중앙 Orchestration 기반 Coding Agent"
```

를 완성하는 것이다.

---

# 172. 이후 확장

Core가 안정된 이후:

```text
Inline Edit

Autocomplete

Agent Planning 개선

Multi-agent Parallel Review

Advanced Security

Cross-project Search

Large Refactoring

Automated Git Workflow
```

를 추가한다.

---

# 173. 최종 MVP Architecture

```text
Developer
   ↓
VS Code
   ↓
Local Agent
   ├─ Workspace
   ├─ Project Intelligence
   ├─ Diff / Apply
   ├─ Maven
   └─ Test
   ↓
onCode Server
   ├─ Workflow
   ├─ Senior Developer
   ├─ Implementation
   ├─ Review
   ├─ Context
   └─ Tool Gateway
   ↓
Internal LLM
```

---

# 174. Pilot Architecture

```text
MVP

+

Guide Repository

Nexus

Security

Authentication

Audit
```

---

# 175. Production Architecture

```text
Pilot

+

HA

Shared Artifact

Monitoring

Policy Management

Backup / Recovery

Upgrade / Rollback
```

---

# 176. 핵심 설계 결정

onCode MVP Scope / Implementation Roadmap v1의 핵심 결정은 다음과 같다.

1. MVP의 목표는 기능 수가 아니라 End-to-End Coding Workflow 완성이다.
2. Local Agent를 Server보다 먼저 구현하여 Local Tool Contract를 안정화한다.
3. Protocol과 Schema를 코드 구현보다 먼저 확정한다.
4. 초기 Agent 수를 최소화한다.
5. 첫 기술 Stack은 Java + Spring Boot + Maven을 권장한다.
6. 첫 IDE는 하나만 지원하고 이후 확장한다.
7. Design Approval과 Code Diff Approval을 MVP 핵심 기능으로 둔다.
8. Local Agent가 실제 Diff를 생성하고 File Hash를 검증한다.
9. Build/Test Fix Loop까지 완료되어야 Technical MVP로 본다.
10. Project Intelligence는 정교한 Call Graph보다 파일 탐색 정확도를 우선한다.
11. Parser/AST 기반 구조 분석을 LLM Summary보다 먼저 구현한다.
12. Guide Ingestion/Retrieval은 Core Coding Loop 이후 통합한다.
13. Vector DB는 MVP에 포함하지 않는다.
14. Nexus/Dependency/Security는 Pilot 단계에서 추가한다.
15. Shell과 위험 Git 기능은 초기 MVP에서 제외한다.
16. Authentication/HA 전체 구현 전에도 Scope ID와 Audit 구조는 초기부터 적용한다.
17. Observability 전체는 후순위라도 Correlation ID와 Structured Error는 P1부터 구현한다.
18. Server Crash, Local Disconnect, Stale Workspace를 정상 기능으로 간주하고 테스트한다.
19. 동일 Tool의 중복 실행 방지를 초기부터 고려한다.
20. Sample Project와 Golden Workflow를 자동화된 Regression Test로 유지한다.
21. MVP, Pilot, Security Pilot, Production을 별도 단계로 관리한다.
22. Production 기능을 MVP에 과도하게 포함하여 Critical Path를 늦추지 않는다.
23. Core Architecture가 안정된 뒤 Agent 역할을 세분화한다.
24. 15개 SPEC 이후에는 새로운 상위 명세보다 ADR/API/DDL/Backlog 문서로 구현 단계에 진입한다.
25. onCode의 최초 성공 기준은 Cursor 전체 복제가 아니라 폐쇄망에서 안전하고 검증 가능한 Agentic Coding Loop의 완성이다.