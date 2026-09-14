# onCode Architecture Decision Records

## 문서 목적

본 문서는 SPEC-01~15에서 정의한 onCode Architecture를 실제 구현 기술로 구체화하기 위한 주요 Architecture Decision Record를 정의한다.

ADR은 다음 원칙을 따른다.

```text
SPEC
→ 시스템이 어떤 구조와 책임을 가져야 하는가

ADR
→ 그 구조를 실제로 어떤 기술과 구현 방식으로 만들 것인가
```

초기 ADR 목록:

```text
ADR-001 Central Server Technology
ADR-002 Local Agent Technology
ADR-003 Server ↔ Local Agent Communication
ADR-004 IDE ↔ Local Agent IPC
ADR-005 Persistence / Session / Distributed State
ADR-006 Project Intelligence Parser Architecture
ADR-007 LLM Inference Architecture
ADR-008 Artifact Storage Architecture
```

---

# ADR-001 Central Server Technology

## 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

onCode 중앙 서버는 다음 기능을 수행한다.

```text
Workflow Orchestration

Agent Runtime

Context Management

Tool Gateway

Authentication / Authorization

Guide Retrieval

Dependency Integration

Persistent Local Agent Connection

Audit / Observability
```

이 중앙 서버의 기본 구현 기술을 결정해야 한다.

## 주요 후보

### A. Java + Spring Boot

장점:

```text
기업 환경 적용 용이

Spring Security

WebSocket

Transaction

Scheduling

PostgreSQL Integration

Observability

LDAP / OAuth / OIDC Integration

장기 운영성
```

단점:

```text
Python 대비 LLM 생태계 직접 활용이 다소 불편

초기 코드량이 상대적으로 많음
```

### B. Python + FastAPI

장점:

```text
LLM / AI Library Integration 용이

빠른 Prototype

Python AI Ecosystem 활용
```

단점:

```text
복잡한 Workflow/Transaction/HA 시스템 운영에서
Java 계열 대비 조직 표준화가 어려울 가능성

대형 서버 Architecture에서 Discipline 필요
```

### C. Node.js / TypeScript

장점:

```text
비동기 네트워크 구현 편리

WebSocket 친화적

개발 생산성
```

단점:

```text
복잡한 Workflow Transaction과 장기 기업 운영에서는
조직 경험에 따라 적합성이 크게 달라짐
```

## 결정

**Java + Spring Boot를 중앙 서버 표준 기술로 권장한다.**

현재 공식 Spring Boot 문서는 4.1.1을 stable release 중 하나로 안내하고 있으며, Spring Boot 자체가 production-grade 애플리케이션을 위한 보안, 메트릭, health check, 외부 설정 등의 기능을 제공하는 방향으로 설계되어 있다.

단, 실제 프로젝트에서는 최신 버전을 무조건 채택하기보다 내부 폐쇄망의 JDK/라이브러리 검증 상태에 따라 지원 가능한 Spring Boot LTS성 안정 버전을 선택한다.

## 권장 구성

```text
Java 21+

Spring Boot

Spring Security

Spring Web / WebSocket

Spring JDBC 또는 JPA

PostgreSQL

Flyway

Micrometer / OpenTelemetry Adapter
```

## 중요한 결정

Agent Runtime 자체도 Spring Boot 내부 모듈로 시작한다.

```text
oncode-server

├─ api
├─ workflow
├─ agent-runtime
├─ context
├─ tool-gateway
├─ guide
├─ dependency
├─ auth
└─ audit
```

초기부터 Agent별 Microservice를 만들지 않는다.

## 결과

```text
Central Server
=
Spring Boot Modular Monolith
```

으로 시작하고, 부하/운영 요구가 발생하면 물리 분리한다.

---

# ADR-002 Local Agent Technology

## 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

Local Agent는 다음 역할을 수행한다.

```text
File Read / Search

Diff / Apply

File Watch

Project Index

Parser

Build

Test

Git

Process Control

IDE IPC

Server Persistent Connection

Local Security Policy
```

Windows, macOS, Linux에서 장시간 실행되는 daemon 형태여야 한다.

## 주요 후보

### A. Java

장점:

```text
Server와 기술 통일

Java/Spring Project 분석 Library 풍부

Cross-platform

Process/File API 안정적
```

단점:

```text
JVM Runtime 필요

Native daemon에 비해 배포 크기 큼

여러 언어 분석 Agent라는 측면에서는 Java 중심성이 강함
```

### B. Go

장점:

```text
Single Binary

Cross Compilation

낮은 Memory Footprint

File/Process/Network 처리 우수

Daemon 구현에 적합

gRPC 지원 우수
```

단점:

```text
JVM AST Library 직접 사용 어려움

일부 Parser Integration에 CGo 필요 가능
```

### C. Rust

장점:

```text
높은 성능

Memory Safety

Native Binary

강력한 시스템 제어
```

단점:

```text
개발 난이도

팀 학습비용

초기 MVP 생산성
```

### D. Python

장점:

```text
구현 속도

Parser/AI Library
```

단점:

```text
배포와 Runtime 관리

Daemon 안정성

Process Packaging

Python Environment 충돌
```

## 결정

**Local Agent는 Go를 권장한다.**

이 선택은 onCode 전체에서 가장 중요한 기술 결정 중 하나다.

Local Agent의 핵심은 AI 연산이 아니라:

```text
Filesystem

Process

Network

Git

Build/Test

Security Boundary

Long-running Daemon
```

이기 때문이다.

Go는 단일 Binary 배포가 가능하고 Windows/macOS/Linux에 동일한 구조로 배포하기 좋다.

## Parser와의 관계

Local Agent 자체가 Go라고 해서 모든 Parser를 Go로 직접 구현하지 않는다.

```text
Local Agent Core (Go)
       ↓
Parser Adapter
       ├─ Tree-sitter
       ├─ Java Semantic Adapter
       └─ Language-specific Adapter
```

구조를 사용한다.

## 대안

팀이 Go 경험이 거의 없고 Java 역량이 압도적으로 높다면:

```text
Java Local Agent
```

도 충분히 현실적인 선택이다.

그러나 기술적으로 새로 선택할 수 있다면 **Go가 Local Agent 역할에 더 적합**하다고 판단한다.

## 결과

```text
Server
→ Java

Local execution boundary
→ Go
```

로 역할에 맞는 기술을 분리한다.

---

# ADR-003 Server ↔ Local Agent Communication

## 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

Server와 Local Agent는 다음 특성을 가져야 한다.

```text
Local Agent → Server outbound connection

Long-lived connection

Bidirectional messaging

Tool request / response

Event

Heartbeat

Cancellation

Reconnect

Large log streaming
```

## 후보

### A. WebSocket + JSON

장점:

```text
단순함

디버깅 편리

기술 중립

Proxy 친화적
```

단점:

```text
Schema enforcement를 별도 구현해야 함

Binary/streaming 구조를 직접 설계

Protocol 관리 부담
```

### B. gRPC Bidirectional Streaming

장점:

```text
Strong Schema

Generated Client/Server

Streaming

Deadline

Cancellation

Metadata

성능
```

gRPC는 한 RPC에서 양쪽이 독립적으로 메시지 Stream을 읽고 쓸 수 있는 bidirectional streaming을 공식 지원한다.

단점:

```text
Protocol Buffers 관리 필요

HTTP/2 운영 고려

일반 Web Debugging보다 복잡
```

## 결정

**Server ↔ Local Agent에는 gRPC Bidirectional Streaming을 권장한다.**

구조:

```text
Local Agent
    │
    │ outbound gRPC stream
    ▼
Server Agent Gateway
```

## 중요한 설계

단, SPEC-01의 논리 Protocol과 gRPC Transport를 동일시하지 않는다.

```text
onCode Tool Protocol
        ↓
Transport Adapter
        ↓
gRPC
```

즉 향후 필요하면 WebSocket Adapter도 추가 가능해야 한다.

## Stream 역할

하나의 Persistent Stream에서:

```text
REGISTER

HEARTBEAT

TOOL_REQUEST

TOOL_RESPONSE

TOOL_EVENT

CANCEL

APPROVAL_EVENT

WORKFLOW_EVENT
```

등을 전달한다.

## 대용량 데이터

Source 전체나 Build Log 전체를 gRPC message 하나에 넣지 않는다.

```text
Chunked Stream
또는
Artifact Reference
```

사용.

## 결과

```text
Logical Protocol
= onCode Tool Protocol

Physical transport
= gRPC bidirectional streaming
```

---

# ADR-004 IDE ↔ Local Agent IPC

## 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

VS Code, IntelliJ, Eclipse Plugin이 Local Agent와 어떻게 통신할지 결정한다.

조건:

```text
localhost only

Simple

IDE language independent

Event Push

Reconnect

Easy debugging
```

## 후보

```text
gRPC

WebSocket

HTTP REST

Named Pipe

Unix Domain Socket
```

## 결정

**localhost WebSocket + JSON을 권장한다.**

Server 통신과 달리 IDE IPC에서는 단순성이 더 중요하다.

구조:

```text
VS Code
     │
IntelliJ
     ├── localhost WebSocket
Eclipse
     │
     ▼
Local Agent
```

## 이유

IDE Plugin 기술은 각각 다르다.

```text
VS Code → TypeScript

IntelliJ → Kotlin/Java

Eclipse → Java
```

JSON/WebSocket은 세 환경에서 구현이 쉽다.

## Security

```text
127.0.0.1 bind only

Random local token

Client registration

Origin validation

Agent restart 시 token rotation
```

적용.

## 향후

고보안 환경에서 필요하면:

```text
Named Pipe
Unix Domain Socket
```

Adapter를 추가할 수 있다.

## 결과

```text
Server ↔ Agent
→ gRPC

IDE ↔ Agent
→ WebSocket + JSON
```

로 서로 다른 목적에 맞게 선택한다.

---

# ADR-005 Persistence / Session / Distributed State

## 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

다음 상태를 어디에 저장할지 결정한다.

```text
Workflow

Context

Work Item

Approval

Project Catalog

Guide

Audit

Session

Connection Registry
```

## 후보

### A. PostgreSQL only

장점:

```text
Architecture 단순

Transaction

Consistency

운영 요소 감소
```

단점:

```text
고속 distributed lock/pub-sub에는 불리
```

### B. PostgreSQL + Redis

장점:

```text
Session Cache

Distributed Lock

Pub/Sub

Connection Registry

빠른 Lookup
```

단점:

```text
운영 구성요소 증가

상태 일관성 복잡성 증가
```

## 결정

**MVP와 Pilot 초기는 PostgreSQL only를 권장한다.**

```text
Official State
→ PostgreSQL
```

사용.

Redis는 요구가 명확해지는 시점까지 도입하지 않는다.

## PostgreSQL 저장 대상

```text
Workflow

Context Metadata

Approval

Session

Project

Guide

Audit

Tool Call
```

## Redis 도입 조건

다음 조건 중 실제 문제가 확인될 때 도입한다.

```text
Persistent Connection 수 급증

Cross-node Routing 비용 증가

Distributed Lock 병목

Session lookup 병목

Pub/Sub 필요
```

## 원칙

Redis를 도입하더라도:

```text
Redis
≠
Source of Truth
```

이다.

공식 Workflow/Approval 상태는 PostgreSQL에 남긴다.

## 결과

```text
MVP:
PostgreSQL only

Scale-out:
PostgreSQL + optional Redis
```

---

# ADR-006 Project Intelligence Parser Architecture

## 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

Java, Python, JavaScript, TypeScript, React, Vue, Spring, eGovFramework 프로젝트에서 다음 정보를 추출해야 한다.

```text
File

Class

Function

Method

Import

Inheritance

Dependency

Framework Metadata

Relation
```

## 후보 1 — 언어별 Native Parser만 사용

예:

```text
Java → JavaParser / JDT

Python → Python AST

TypeScript → TypeScript Compiler API
```

장점:

```text
Semantic 정보 풍부
```

단점:

```text
언어별 구현 방식이 크게 달라짐

Maintenance 증가
```

## 후보 2 — Tree-sitter 중심

Tree-sitter는 incremental parsing library로 설계되어 있고, 소스가 수정될 때 기존 syntax tree를 활용해 효율적으로 갱신할 수 있으며 syntax error가 있는 코드에서도 유용한 parsing 결과를 제공하는 것을 목표로 한다.

지원 언어도 Java, Python, JavaScript, TypeScript 등 onCode가 필요한 주요 언어를 폭넓게 포함한다.

장점:

```text
다중 언어 공통화

Incremental Parsing

빠른 속도

Broken Source 대응

Local Editor Workflow에 적합
```

단점:

```text
Concrete Syntax 중심

Type Resolution 부족

Semantic Resolution 한계
```

## 결정

**Tree-sitter를 공통 Structural Parser로 사용하고, 필요할 때 Language-specific Semantic Adapter를 추가하는 Hybrid 방식을 권장한다.**

구조:

```text
Project Intelligence
       ↓
Parser Registry
       │
       ├─ Tree-sitter Java
       ├─ Tree-sitter Python
       ├─ Tree-sitter JS/TS
       ├─ Tree-sitter HTML/Vue
       │
       └─ Semantic Adapter
             ├─ Java
             └─ TypeScript
```

## 1단계

Tree-sitter로:

```text
Class

Method

Function

Import

Annotation

Basic Call

File structure
```

추출.

## 2단계

Java에서 필요해지면:

```text
JavaParser + Symbol Solver
또는
Eclipse JDT
```

등을 추가한다.

TypeScript 역시 정확한 type resolution이 필요할 때 TypeScript Language Service/Compiler Adapter를 추가한다.

## Spring/eGovFramework

Framework 분석은 Parser 자체에 넣지 않는다.

```text
AST
 ↓
Framework Analyzer
```

예:

```text
@RestController

@Service

@Repository

@RequestMapping
```

을 별도 Framework Analyzer가 해석한다.

## 결과

```text
Structural truth
→ Tree-sitter

Deep semantic truth
→ language-specific adapters
```

---

# ADR-007 LLM Inference Architecture

## 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

여러 Agent가 폐쇄망 내부 LLM을 안정적으로 호출해야 한다.

Agent가 특정 Model Endpoint에 직접 연결하면 다음 문제가 생긴다.

```text
Model 변경 어려움

Concurrency 관리 어려움

GPU 자원 관리 분산

Fallback 어려움

Metrics 분산
```

## 결정

다음 구조를 사용한다.

```text
Agent Runtime
      ↓
Inference Gateway
      ↓
vLLM Server Pool
```

**Inference Engine은 vLLM을 우선 권장하고, Agent는 OpenAI-compatible internal API만 사용한다.**

vLLM은 현재 Chat Completions, Responses, Embeddings 등을 포함한 OpenAI-compatible HTTP API를 제공한다.

## Agent Model Profile

Agent에는 모델 이름이 아니라 Profile을 지정한다.

```text
reasoning-high

coding

review

utility
```

예:

```text
Senior Developer
→ reasoning-high

Implementation
→ coding

Guide Query Analyzer
→ utility
```

## Inference Gateway 책임

```text
profile → model routing

timeout

queue

concurrency

token limit

fallback

metrics

health
```

## 중요한 보안 결정

vLLM endpoint를 개발자 PC나 Agent에 직접 노출하지 않는다.

```text
Agent Runtime
→ Inference Gateway
→ vLLM
```

으로 제한한다.

또한 vLLM 공식 문서는 API key 옵션만으로 모든 endpoint가 보호되는 것은 아니므로 reverse proxy 등 별도 보안 계층으로 hardening할 것을 안내하고 있다. 따라서 폐쇄망이라고 해도 vLLM 자체 endpoint를 신뢰 경계 외부에 직접 노출하지 않는다.

## MVP

처음에는:

```text
Inference Gateway

→ 1 coding/reasoning model
```

만 있어도 된다.

향후:

```text
large reasoning model

coding model

small classification model
```

로 분리.

## 결과

```text
Agent
≠
Model

Agent
→ Model Profile
→ Inference Gateway
→ vLLM
```

---

# ADR-008 Artifact Storage Architecture

## 상태

```text
PROPOSED → RECOMMENDED
```

## 결정할 문제

다음 대용량 Artifact를 저장해야 한다.

```text
Build Log

Test Log

Large Diff

Guide Original

Converted PDF

Markdown

Docling JSON

Temporary Source Snapshot
```

이 데이터를 PostgreSQL에 직접 넣는 것은 적절하지 않다.

## 후보

### A. Local Filesystem

장점:

```text
단순
```

단점:

```text
HA 불가

Server node 종속
```

### B. NFS/NAS

장점:

```text
기존 기업 인프라 활용

운영 단순

Shared Storage
```

단점:

```text
Object lifecycle/API 기능 제한
```

### C. S3-compatible Object Storage

장점:

```text
Object abstraction

Retention

Scalability

HA

Application node와 분리
```

단점:

```text
추가 운영 인프라
```

## 결정

**MVP에서는 NFS/NAS-compatible shared storage를 권장하고, Application에는 ArtifactStorage 인터페이스를 둔다.**

즉:

```text
Application
      ↓
ArtifactStorage
      ↓
NFS
```

형태.

## 중요 원칙

Application 코드에서 물리 파일 경로를 직접 사용하지 않는다.

```text
artifact://build/EXEC-100/log

artifact://guide/DOC-100/original
```

처럼 논리 URI를 사용한다.

## 인터페이스

```text
put()

get()

delete()

exists()

metadata()
```

## 향후

운영 규모가 커지면:

```text
S3ArtifactStorage
```

를 구현하여 MinIO 등의 내부 S3-compatible storage로 교체할 수 있다.

## Local Agent Artifact

Local Agent가 생성하는:

```text
temporary diff

backup

raw build log
```

는 먼저 Local Storage에 두고 필요한 것만 중앙 Artifact Storage로 올린다.

## 결과

```text
MVP:
Shared NFS/NAS

Architecture:
ArtifactStorage abstraction

Future:
Internal S3-compatible storage
```

---

# 9. 최종 권장 기술 조합

8개 ADR을 종합하면 다음 구성이 된다.

```text
                    onCode Architecture


IDE Extension
    │
    │ WebSocket + JSON
    ▼
Local Agent
    │
    │ Go
    │
    ├─ Tree-sitter
    ├─ Workspace
    ├─ Git
    ├─ Build/Test
    └─ Policy
    │
    │ gRPC Bidirectional Streaming
    ▼
onCode Central Server
    │
    │ Java / Spring Boot
    │
    ├─ Orchestrator
    ├─ Agent Runtime
    ├─ Context
    ├─ Tool Gateway
    ├─ Guide
    ├─ Dependency
    └─ Security
    │
    ├──────────────► PostgreSQL
    │
    ├──────────────► Shared Artifact Storage
    │
    ├──────────────► Nexus
    │
    └──────────────► Inference Gateway
                              │
                              ▼
                            vLLM
```

---

# 10. 권장 기술 선택 요약

| 영역 | 권장안 |
|---|---|
| Central Server | **Java + Spring Boot** |
| Server 구조 | **Modular Monolith 우선** |
| Local Agent | **Go** |
| Server ↔ Local Agent | **gRPC Bidirectional Streaming** |
| IDE ↔ Local Agent | **localhost WebSocket + JSON** |
| Main DB | **PostgreSQL** |
| Redis | **MVP 제외, 필요 시 추가** |
| Structural Parser | **Tree-sitter** |
| Deep Semantic Parser | **Language-specific Adapter** |
| Inference Gateway | **onCode 자체 Gateway** |
| LLM Serving | **vLLM** |
| Artifact | **NFS/NAS → 향후 S3-compatible** |
| Guide Search | **PostgreSQL FTS/Keyword/Trigram** |
| Package Source | **기존 Nexus** |
| External Internet | **Runtime 사용 금지** |

---

# 11. 특히 중요한 세 가지 ADR

전체 중 개발 난이도와 이후 변경 비용이 가장 큰 것은 다음 세 가지다.

## 1. ADR-002 Local Agent = Go

Local Agent는 onCode의 실제 실행 경계다.

따라서 이 선택은:

```text
Cross-platform

배포

Parser

Git

Process

Build

Security
```

전체에 영향을 준다.

---

## 2. ADR-003 Server ↔ Local Agent = gRPC

이 Protocol은 거의 모든 기능이 지나간다.

```text
source read

search

diff

build

test

git

event

progress
```

따라서 초기에 제대로 설계해야 한다.

---

## 3. ADR-006 Tree-sitter + Semantic Adapter

이 결정은 Project Intelligence 품질과 지원 언어 확장 비용을 결정한다.

모든 언어에 완벽한 Semantic Parser를 처음부터 구현하는 대신:

```text
Tree-sitter
→ 공통 구조 분석

Semantic Adapter
→ 필요한 곳만 강화
```

가 onCode MVP에 가장 현실적인 전략이다.

---

# 12. 구현 전 추가 ADR 권장

위 8개가 결정되면 바로 코드에 들어갈 수도 있지만 다음 네 가지는 추가 ADR로 만드는 것이 좋다.

```text
ADR-009 Repository Strategy
Monorepo vs Multi-repo

ADR-010 Workflow Execution Model
DB polling vs internal queue vs scheduler

ADR-011 Change Application Model
Full-file replacement vs structured patch

ADR-012 Project Search Architecture
PostgreSQL FTS vs Local index + central catalog
```

특히 **ADR-011은 매우 중요하다.**

LLM이:

```text
전체 수정 파일을 반환할지

Unified Diff를 반환할지

Structured Edit Operation을 반환할지
```

에 따라 Local Agent Apply 구조와 안정성이 크게 달라지기 때문이다.

현재 onCode Architecture에는 저는 다음 방식을 추천합니다.

```text
Implementation Agent
    ↓
Structured ProposedChanges
    ↓
Local Agent
    ↓
Current File + Proposed Content
    ↓
Actual Diff 생성
    ↓
Developer Approval
    ↓
Hash Check
    ↓
Atomic Apply
```

즉 **LLM이 만든 Unified Diff를 그대로 적용하지 않는 구조**입니다. 이 부분은 다음 ADR에서 별도로 확정하는 것이 좋습니다.