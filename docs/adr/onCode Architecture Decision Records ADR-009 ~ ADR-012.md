# onCode Architecture Decision Records

## ADR-009 Repository Strategy

### 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

onCode는 다음과 같이 기술 스택이 여러 개다.

```text
Central Server
→ Java / Spring Boot

Local Agent
→ Go

IDE Extension
→ TypeScript / Kotlin / Java

Guide Ingestion
→ Python / Docling / LibreOffice

Deployment
→ Container / YAML / Script
```

이들을 하나의 Repository에서 관리할지, 기능별 Multi-repo로 분리할지 결정해야 한다.

---

## 후보 A. Monorepo

예:

```text
oncode/

├─ server/
├─ local-agent/
├─ ide/
│  ├─ vscode/
│  ├─ intellij/
│  └─ eclipse/
├─ protocol/
├─ guide-ingestion/
├─ guide-admin/
├─ deployment/
├─ schemas/
└─ samples/
```

장점:

```text
Protocol 변경 추적 용이

Server / Local Agent 동시 변경 쉬움

Cross-component PR 관리

Sample / Contract Test 공유

초기 Architecture 변화 대응 용이
```

단점:

```text
Repository 규모 증가

여러 Build Tool 공존

CI Pipeline 복잡
```

---

## 후보 B. Multi-repo

예:

```text
oncode-server

oncode-local-agent

oncode-vscode

oncode-guide-ingestion

oncode-protocol
```

장점:

```text
컴포넌트 독립 배포

권한 분리

Repository 크기 작음

기술별 CI 단순
```

단점:

```text
Protocol 변경 시 여러 Repository 동기화 필요

Version mismatch 위험

초기 개발 속도 저하
```

---

## 결정

**초기에는 Monorepo를 권장한다.**

onCode MVP 단계에서는 Protocol, Schema, Workflow, Local Tool 계약이 자주 변경될 가능성이 높다.

따라서 다음이 중요하다.

```text
한 Commit에서

Protocol Schema 변경
+
Server 변경
+
Local Agent 변경
+
Contract Test 변경
```

을 함께 관리할 수 있어야 한다.

---

## 권장 구조

```text
oncode/

├─ server/
│  ├─ api/
│  ├─ workflow/
│  ├─ agent-runtime/
│  ├─ context/
│  ├─ tool-gateway/
│  └─ security/
│
├─ local-agent/
│
├─ ide/
│  ├─ vscode/
│  ├─ intellij/
│  └─ eclipse/
│
├─ protocol/
│  ├─ proto/
│  ├─ json-schema/
│  └─ generated/
│
├─ guide/
│  ├─ ingestion/
│  └─ admin-web/
│
├─ deployment/
│
├─ samples/
│
├─ docs/
│  ├─ spec/
│  └─ adr/
│
└─ tests/
   ├─ contract/
   └─ e2e/
```

---

## Protocol 생성 코드

가능하면:

```text
protocol/proto/
```

에서 gRPC Stub을 생성한다.

Server:

```text
Java generated client/server
```

Local Agent:

```text
Go generated client/server
```

를 같은 Protocol Source에서 생성한다.

---

## 분리 시점

다음 조건이 생기면 Multi-repo 분리를 검토한다.

```text
배포 주기가 완전히 달라짐

팀 Ownership이 완전히 분리

Repository 크기/CI 시간이 문제

보안상 접근 권한 분리가 필요

Protocol이 충분히 안정화됨
```

---

## 결과

```text
MVP / Pilot
→ Monorepo

Production 이후
→ 필요 시 컴포넌트별 분리
```

---

# ADR-010 Workflow Execution Model

### 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

Workflow Orchestrator가 다음 작업을 어떤 실행 모델로 처리할지 결정해야 한다.

```text
Agent Task 실행

Tool 요청

Approval 대기

User Input 대기

Retry

Timeout

Resume

Server 장애 후 복구
```

---

## 후보 A. 서버 메모리 기반 State Machine

장점:

```text
구현 간단

빠름
```

단점:

```text
Server 재시작 시 상태 손실

HA 어려움

Resume 어려움
```

부적합하다.

---

## 후보 B. Message Broker 중심

예:

```text
Kafka

RabbitMQ

Redis Streams
```

장점:

```text
비동기 처리 강함

Scale-out

Worker 분리
```

단점:

```text
MVP 복잡성 증가

Workflow 상태와 Event 상태 이중 관리

운영 요소 증가
```

---

## 후보 C. PostgreSQL 기반 Persistent State Machine

구조:

```text
WORKFLOW
+
WORKFLOW_TRANSITION
+
TASK
+
Lease / Optimistic Lock
```

Worker가 실행 가능한 Task를 가져가 처리한다.

장점:

```text
State 일관성

Transaction

Resume

운영 단순

Audit와 연계 쉬움
```

단점:

```text
초대규모 Queue 처리에는 한계
```

---

## 결정

**MVP와 Pilot에서는 PostgreSQL 기반 Persistent Workflow Engine을 권장한다.**

구조:

```text
API
 ↓
WORK_ITEM 생성
 ↓
WORKFLOW State 저장
 ↓
Workflow Executor
 ↓
Next Task
 ↓
Agent / Tool
 ↓
Result
 ↓
State Transition
```

---

## Workflow Executor

```text
Workflow Scheduler
    ↓
Runnable Workflow 검색
    ↓
Lease 획득
    ↓
State Handler 실행
    ↓
Transition 저장
    ↓
Lease Release
```

---

## State Handler

예:

```text
DESIGNING
→ DesignHandler

BUILDING
→ BuildHandler

WAITING_APPROVAL
→ ApprovalHandler
```

가능하면 상태별 처리를 명시적인 Handler로 구현한다.

---

## DB Polling

MVP에서는 단순한 DB polling도 허용한다.

예:

```text
200~500 ms
```

주기로 runnable task 조회.

단, busy loop는 피한다.

---

## Event Wake-up

향후 성능이 필요하면:

```text
DB = Source of Truth
+
Redis / Broker = Wake-up Signal
```

구조로 확장한다.

즉 Broker를 공식 상태 저장소로 사용하지 않는다.

---

## 상태 변경 Transaction

예:

```text
BUILDING
 ↓
build.run 호출 생성
```

시 다음을 Transaction으로 묶는 것이 좋다.

```text
Workflow Revision Update

Tool Call Record

Outbox Event
```

---

## Workflow Lock

DB Row Lock에 장시간 의존하지 않는다.

대신:

```text
workflow_revision

lease_owner

lease_until
```

사용.

---

## Lease 예

```text
lease_owner = server-2
lease_until = now + 30s
```

처리가 길면 연장한다.

---

## Idempotency

Workflow Task마다:

```text
task_id

attempt
```

가 존재한다.

동일 Task 재실행 시 이전 결과를 확인한다.

---

## WAITING 상태

다음 상태는 Worker를 소비하지 않는다.

```text
WAITING_USER_INPUT

WAITING_APPROVAL

WAITING_LOCAL_AGENT
```

외부 Event가 발생하면 runnable 상태로 전환한다.

---

## 결과

```text
Official workflow state
→ PostgreSQL

Execution
→ State Handler + Lease

Scale-out signal
→ 향후 Redis/Broker 추가 가능
```

---

# ADR-011 Change Application Model

### 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

Implementation Agent가 생성한 변경사항을 Local Workspace에 어떤 방식으로 전달하고 적용할지 결정해야 한다.

후보:

```text
Full File Replacement

Unified Diff

Search/Replace Patch

AST Edit

Structured File Change
```

이 결정은 onCode 안전성에 직접 영향을 준다.

---

## 후보 A. LLM이 Unified Diff 생성

예:

```diff
--- a/AuthService.java
+++ b/AuthService.java
@@ ...
```

장점:

```text
변경량 적음

일반 개발 도구와 친숙
```

단점:

```text
Context mismatch

Line drift

Malformed diff

Patch 적용 실패

LLM-generated diff 자체 신뢰 문제
```

---

## 후보 B. Full File Replacement

Agent가 전체 파일 최종 내용을 반환한다.

장점:

```text
구현 단순

적용 deterministic
```

단점:

```text
큰 파일 Context 비용

불필요한 변경 위험

다른 사용자 변경 덮어쓰기 위험
```

---

## 후보 C. Structured Proposed Change

예:

```json
{
  "changes": [
    {
      "path": "src/main/java/AuthService.java",
      "operation": "MODIFY",
      "base_hash": "...",
      "content": "..."
    }
  ]
}
```

Local Agent가 현재 Workspace와 비교해 실제 Diff를 만든다.

장점:

```text
Server 제안과 Local 실제 상태 분리

Stale Detection 명확

IDE Diff 신뢰 가능

Atomic Apply 가능
```

단점:

```text
수정 파일 전체 content가 필요할 수 있음
```

---

## 후보 D. Structured Edit Operations

예:

```json
{
  "path": "...",
  "edits": [
    {
      "range": {...},
      "replacement": "..."
    }
  ]
}
```

장점:

```text
전송량 적음

부분 수정에 적합
```

단점:

```text
Range drift

Source revision 의존성 강함

LLM이 정확한 range를 생성해야 함
```

---

## 결정

**v1에서는 Structured Proposed Change + Local Diff Generation을 권장한다.**

핵심 원칙:

```text
LLM Diff
≠
승인 대상 Diff

승인 대상 Diff
=
Local Agent가 현재 Workspace와 비교해 만든 실제 Diff
```

---

## Change Set 구조

```json
{
  "change_set_id": "CHG-100",
  "base_workspace_revision": 52,
  "changes": [
    {
      "path": "src/main/java/AuthService.java",
      "operation": "MODIFY",
      "base_hash": "sha256:...",
      "proposed_content_ref": "..."
    }
  ]
}
```

---

## Content 전달 방식

작은 파일:

```text
inline content
```

큰 파일:

```text
artifact reference
```

가능.

---

## CREATE

```json
{
  "operation": "CREATE",
  "path": "src/main/java/AccountLockPolicy.java",
  "content": "..."
}
```

---

## MODIFY

```text
base_hash 필수
```

---

## DELETE

```text
path

base_hash
```

필수.

---

## RENAME

```text
source_path

target_path

source_hash
```

필수.

---

## Local Diff

```text
Proposed Content
+
Current Local File
        ↓
Local Agent Diff Engine
        ↓
Unified Diff
        ↓
IDE Native Diff
```

---

## Approval

승인 대상:

```text
change_set_id

base hashes

actual diff hash
```

와 연결한다.

---

## Apply

승인 후 다시:

```text
current hash == approved base hash
```

검증.

다르면:

```text
WORKSPACE_FILE_CHANGED
```

---

## Atomicity

기본:

```text
All file preconditions valid
→ apply all

Any invalid
→ apply none
```

---

## Partial Approval

v1에서는 지원하지 않는다.

후속 기능으로 둔다.

---

## AST Editing

AST Edit는 특정 언어의 정교한 Refactoring에 유용하지만 v1의 일반 변경 모델로 사용하지 않는다.

향후:

```text
Structured Refactor Operation
```

으로 별도 도입할 수 있다.

---

## Full File Replacement 관련 결정

Implementation Agent 내부적으로는 전체 최종 파일을 생성해도 된다.

다만 사용자에게는:

```text
전체 파일
```

이 아니라:

```text
Local actual diff
```

만 보여준다.

---

## 결과

```text
Implementation Agent
→ Structured Proposed Change Set

Local Agent
→ Actual Diff

User
→ Approve Actual Diff

Local Agent
→ Hash Validate + Atomic Apply
```

---

# ADR-012 Project Search Architecture

### 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

Server Agent가 개발 요청과 관련된 Project File/Symbol을 어떻게 찾을지 결정해야 한다.

조건:

```text
폐쇄망

Vector DB 없이 시작

Source 전체 중앙 저장 최소화

Incremental Index

다중 언어

Project별 isolation
```

---

## 후보 A. 중앙 DB에 모든 Source 저장 후 검색

장점:

```text
검색 단순
```

단점:

```text
Source 중앙 집중

Security 위험

동기화 비용

Workspace stale 문제
```

onCode 원칙과 맞지 않는다.

---

## 후보 B. 모든 검색을 Local Agent에서 수행

```text
Server query
→ Local Agent
→ Search
```

장점:

```text
최신 Workspace 기준

Source 중앙 저장 불필요
```

단점:

```text
매 검색마다 Local Round-trip

Central Agent planning에 비효율

Offline workspace 검색 불가

Project-wide catalog 활용 어려움
```

---

## 후보 C. Hybrid

```text
Local Agent
→ PROJECT_INDEX 생성
→ Central Project Catalog 동기화

Server
→ Central Catalog Search

필요한 실제 Source
→ Local Agent Read
```

장점:

```text
검색 빠름

Source 자체 중앙 저장 최소화

Workspace structural state 활용

Agent Context 최소화
```

단점:

```text
Index Sync 필요

Stale 관리 필요
```

---

## 결정

**Hybrid Search Architecture를 권장한다.**

구조:

```text
Local Workspace
   ↓
Project Intelligence
   ↓
PROJECT_INDEX
   ↓
Delta Sync
   ↓
Central Project Catalog
   ↓
project.search
   ↓
Relevant File/Symbol
   ↓
workspace.read_files
   ↓
Actual Source
```

---

## Central에 저장하는 것

```text
File path

File hash

Language

Module

Symbol

Signature

Summary

Dependency

Framework metadata

Relations
```

---

## Central에 기본 저장하지 않는 것

```text
전체 Source

전체 Config value

Secret

Binary
```

---

## Search 단계

### 1. Exact Search

```text
class name

method name

file path

annotation

API
```

최우선.

### 2. Keyword Search

```text
title

summary

qualified name

signature
```

### 3. PostgreSQL Full Text Search

Project Catalog에서 수행.

### 4. Trigram

오타/부분 이름.

### 5. Relation Expansion

```text
Controller
→ Service
→ Repository
```

depth 1 정도.

---

## Ranking 예

```text
exact symbol match

file name match

summary match

framework match

relation proximity

module relevance
```

를 합산.

---

## Search Result

```json
{
  "results": [
    {
      "file": "src/main/java/AuthService.java",
      "symbol": "AuthService.login",
      "score": 0.95,
      "reason": [
        "symbol name match",
        "summary match",
        "Spring service"
      ]
    }
  ]
}
```

---

## 실제 구현 전 Source Read

중요 규칙:

```text
Project Index
→ Discovery

Actual Source
→ Implementation Truth
```

따라서 Implementation 전에 관련 실제 Source를 Local Agent에서 반드시 읽는다.

---

## Stale Index

Server Catalog Revision:

```text
52
```

Local Workspace Revision:

```text
55
```

이면:

```text
PROJECT_INDEX_STALE
```

또는 Delta Sync 후 검색한다.

---

## Search 요청 시 Revision

```json
{
  "project_id": "PRJ-100",
  "workspace_id": "WS-100",
  "expected_revision": 55
}
```

처럼 명확히 할 수 있다.

---

## Offline Workspace

Local Agent가 Offline이라도 Catalog에서 과거 구조 검색은 가능할 수 있다.

하지만:

```text
actual implementation
```

은 금지한다.

---

## Vector DB

v1에서는 도입하지 않는다.

필요하면 향후:

```text
Project Search Provider
```

인터페이스 뒤에 Hybrid Semantic Search를 추가할 수 있다.

---

## Project Search Provider

```text
ProjectSearchProvider

searchFiles()

searchSymbols()

expandRelations()
```

구현:

```text
PostgresProjectSearchProvider
```

---

## 검색 품질 평가

Golden Query를 유지한다.

예:

```text
Query:
"로그인 처리"

Expected:
LoginController
AuthService
```

KPI:

```text
Top-5 relevant file recall
```

측정.

---

## 결과

```text
Local Agent
→ structural index source

Central PostgreSQL
→ search catalog

Local Agent
→ actual source provider
```

---

# ADR-009 ~ ADR-012 결정 요약

| ADR | 결정 | 권장안 |
|---|---|---|
| ADR-009 | Repository 전략 | **초기 Monorepo** |
| ADR-010 | Workflow 실행 | **PostgreSQL Persistent State Machine + Lease** |
| ADR-011 | 코드 변경 모델 | **Structured Change Set + Local Actual Diff + Atomic Apply** |
| ADR-012 | Project 검색 | **Local Index + Central Catalog + Local Source Read Hybrid** |

---

# 전체 ADR 기술 조합

```text
IDE
 │
 │ localhost WebSocket + JSON
 ▼
Local Agent
 │
 │ Go
 │
 ├─ Tree-sitter
 ├─ Project Index
 ├─ Workspace
 ├─ Diff / Atomic Apply
 ├─ Git
 └─ Build/Test
 │
 │ gRPC Bidirectional Stream
 ▼
Central Server
 │
 │ Java / Spring Boot
 │
 ├─ PostgreSQL Persistent Workflow
 ├─ Context Storage
 ├─ Central Project Catalog
 ├─ Tool Gateway
 ├─ Agent Runtime
 └─ Guide / Dependency / Security
 │
 ├── PostgreSQL
 ├── NFS/NAS Artifact Storage
 ├── Existing Nexus
 └── Inference Gateway
           │
           ▼
          vLLM
```

---

# ADR 최종 목록

현재 구현 착수 전에 필요한 핵심 ADR은 다음 12개로 보는 것이 적절하다.

```text
ADR-001 Central Server Technology
ADR-002 Local Agent Technology
ADR-003 Server ↔ Local Agent Communication
ADR-004 IDE ↔ Local Agent IPC
ADR-005 Persistence / Session / Distributed State
ADR-006 Project Intelligence Parser Architecture
ADR-007 LLM Inference Architecture
ADR-008 Artifact Storage Architecture
ADR-009 Repository Strategy
ADR-010 Workflow Execution Model
ADR-011 Change Application Model
ADR-012 Project Search Architecture
```

이 정도면 상위 아키텍처 선택은 충분하다.

이후부터는 ADR을 계속 늘리기보다 실제 구현 산출물로 내려가는 것이 좋다.

```text
ADR
 ↓
Protocol Definition
 ↓
JSON Schema / Protobuf
 ↓
Database DDL
 ↓
Module Interface
 ↓
Sequence Diagram
 ↓
Epic / Story
 ↓
Implementation
```