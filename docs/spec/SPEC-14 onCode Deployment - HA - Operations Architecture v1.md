# SPEC-14 onCode Deployment / HA / Operations Architecture v1

## 1. 목적

본 명세는 onCode의 배포(Deployment), 고가용성(High Availability), 확장성(Scalability), 장애복구(Recovery), 운영(Operations) 아키텍처를 정의한다.

onCode는 다음 특성을 가진다.

- 중앙 서버 기반 Orchestration
- 다중 사용자 / 다중 프로젝트
- 개발자 PC의 Local Agent와 Persistent Connection
- 폐쇄망 내부 LLM 추론
- PostgreSQL 기반 Context / Workflow / Guide Repository
- 내부 Nexus
- Offline Vulnerability DB
- Guide Ingestion Worker
- Build/Test/Git은 Local Agent에서 실행
- Human-in-the-Loop 승인 기반 Workflow

따라서 일반적인 단일 Web Application 배포보다 상태 관리, 세션 연결, Local Agent 라우팅, LLM 자원 관리, 장애복구 전략이 중요하다.

---

# 2. 배포 목표

주요 목표는 다음과 같다.

1. 중앙 서버의 단일 장애점을 최소화한다.
2. Local Agent 연결이 특정 서버 인스턴스에 과도하게 종속되지 않도록 한다.
3. Workflow 상태를 서버 프로세스 메모리에만 보관하지 않는다.
4. Agent Runtime과 LLM 추론 자원을 분리한다.
5. Guide Ingestion과 Runtime 요청 처리를 자원적으로 분리한다.
6. 서버 장애 후 Workflow Resume가 가능해야 한다.
7. 폐쇄망 환경에서 외부 인터넷 없이 설치/업데이트 가능해야 한다.
8. 내부 Nexus, 인증, Git 등 기존 인프라를 재사용한다.
9. 서비스별 독립 확장이 가능해야 한다.
10. 초기 MVP는 지나친 Microservice 복잡성을 피한다.

---

# 3. 전체 배포 구조

권장 논리 구조:

```text
                       ┌───────────────────┐
                       │   IDE Extensions  │
                       └─────────┬─────────┘
                                 │
                                 ▼
                       ┌───────────────────┐
                       │    Local Agent    │
                       └─────────┬─────────┘
                                 │
                        Persistent TLS
                                 │
                                 ▼
                    ┌─────────────────────────┐
                    │ Load Balancer / Gateway │
                    └────────────┬────────────┘
                                 │
                ┌────────────────┼─────────────────┐
                ▼                ▼                 ▼
        onCode Server #1  onCode Server #2  onCode Server #N
                │                │                 │
                └────────────────┼─────────────────┘
                                 │
          ┌──────────────────────┼────────────────────────┐
          ▼                      ▼                        ▼
   PostgreSQL Cluster      Inference Gateway       Internal Services
                                                   ├─ Nexus
                                                   ├─ Vulnerability DB
                                                   ├─ LDAP / IdP
                                                   ├─ Git
                                                   └─ Guide Services
```

---

# 4. 초기 배포 단위

초기에는 하나의 onCode Server Application 안에 다음을 함께 배포할 수 있다.

```text
onCode Server

├─ API
├─ Session Service
├─ Workflow Orchestrator
├─ Agent Runtime
├─ Tool Gateway
├─ Context Service
├─ Project Context Service
├─ Guide Retrieval Service
├─ Dependency Service
├─ Authorization Service
└─ Audit Adapter
```

이들은 논리적으로 분리하지만 반드시 각각 별도 Microservice일 필요는 없다.

---

# 5. Microservice 분리 기준

다음 조건이 발생할 때 물리 분리를 검토한다.

```text
부하 특성이 크게 다름

독립 Scale-out 필요

장애 격리가 중요

독립 배포 주기가 다름

보안 경계가 다름

기술 Stack이 다름
```

---

# 6. 우선 분리 후보

물리적 분리 우선순위:

```text
1. Inference Gateway
2. Guide Ingestion
3. Worker / Background Job
4. Observability
5. Runtime Guide Search
```

Orchestrator와 Agent Runtime은 초기에는 같은 서버 Application에 두는 것이 단순하다.

---

# 7. Stateless Server 원칙

onCode Server 인스턴스는 가능한 한 Stateless하게 유지한다.

서버 메모리에만 두지 않는 정보:

```text
Workflow State

Session State

Pending Approval

Context

Tool Call

Agent Result

Project Catalog

Guide Reference
```

이 정보는 중앙 저장소에 기록한다.

---

# 8. Stateful 데이터

공식 상태 저장:

```text
PostgreSQL

Context Storage
Workflow State
Project Catalog
Guide Runtime
Audit Metadata
Session Metadata
```

대용량:

```text
Artifact Storage
```

사용.

---

# 9. Server 재시작

Server Process가 재시작해도 다음에서 상태를 복구한다.

```text
Workflow DB

Context Storage

Session Store

Pending Tool Calls

Pending Approval
```

Agent 자체 실행 Context를 복구하는 것이 아니라 새 Agent 실행으로 Resume한다.

---

# 10. Runtime Server HA

Production에서는 최소:

```text
2 instances
```

이상 운영을 권장한다.

```text
Load Balancer
 ├─ Server A
 └─ Server B
```

한 인스턴스 장애 시 다른 인스턴스가 신규 요청을 처리한다.

---

# 11. HTTP/API 요청

일반 HTTP 요청은 Stateless Routing을 사용한다.

Sticky Session은 기본적으로 필요하지 않다.

---

# 12. Local Agent Persistent Connection

WebSocket/gRPC Streaming과 같은 Persistent Connection은 특별 처리가 필요하다.

연결 자체는 특정 서버 인스턴스에 붙지만 논리 Session은 중앙 Store에서 관리한다.

---

# 13. Connection Registry

서버는 별도 Connection Registry를 유지한다.

예:

```text
local_agent_id
session_id
connected_server_id
connection_id
last_heartbeat
status
```

---

# 14. Local Agent Routing

Tool Gateway가 Local Agent Tool을 호출하려면 현재 연결된 Server Instance를 알아야 한다.

구조:

```text
Tool Gateway
    ↓
Connection Registry
    ↓
Server Instance B
    ↓
Local Agent
```

---

# 15. Cross-node Dispatch

Tool을 요청한 Workflow가 Server A에 있고 Local Agent 연결은 Server B에 있을 수 있다.

따라서 Server 간 Dispatch Channel이 필요하다.

예:

```text
Message Broker

Redis Pub/Sub

Internal gRPC

Database Queue
```

---

# 16. 권장 방식

초기 HA 환경에서는 Durable Message Broker 또는 Redis-compatible messaging을 고려할 수 있다.

핵심은:

```text
Tool request
→ connected node
→ Local Agent
→ result
→ Workflow owner
```

상관관계를 유지하는 것이다.

---

# 17. Workflow Ownership

Workflow를 특정 Server Instance가 독점적으로 장기 소유하지 않도록 한다.

DB 기반 Lease 또는 Distributed Lock을 사용한다.

---

# 18. Workflow Lease

예:

```text
workflow_id

worker_id

lease_until

revision
```

Worker가 단계 실행 중 Lease를 획득한다.

---

# 19. Server 장애 시 Workflow Recovery

Server A 장애:

```text
Lease timeout
   ↓
Server B
   ↓
Workflow State Load
   ↓
Pending State 확인
   ↓
Resume
```

---

# 20. 중복 실행 방지

HA 환경에서는 같은 Workflow Task가 두 번 실행되지 않도록 해야 한다.

방법:

```text
Optimistic Locking

Lease

Idempotency Key

Tool Call Status
```

조합.

---

# 21. Idempotency

특히 다음 작업은 중요하다.

```text
workspace.apply_changes

git.commit

git.push

dependency.install
```

Idempotency Key 또는 실행 결과 확인을 적용한다.

---

# 22. PostgreSQL

onCode의 기본 RDBMS로 PostgreSQL을 권장한다.

용도:

```text
Identity Metadata

Workflow

Context

Project Catalog

Guide Repository

Audit

Operational Metadata
```

---

# 23. Database Schema 분리

예:

```text
identity.*

project.*

workflow.*

context.*

guide_runtime.*

guide_ingestion.*

dependency.*

audit.*

operation.*
```

---

# 24. PostgreSQL HA

Production에서는 다음 중 조직 표준 방식을 사용한다.

```text
Primary + Standby

Patroni 계열

Managed Internal PostgreSQL Cluster

DB Appliance
```

특정 제품을 강제하지 않는다.

---

# 25. DB 장애

DB가 없으면 공식 Workflow 상태를 안전하게 유지할 수 없으므로 Runtime Server는:

```text
DEGRADED 또는 NOT_READY
```

상태로 전환한다.

신규 개발 작업은 중단한다.

---

# 26. DB Failover 후

Application은:

```text
Connection Pool reconnect

Pending Transaction 확인

Workflow Lease 재획득
```

으로 정상화한다.

---

# 27. Artifact Storage

저장 대상:

```text
Build Log

Test Log

Large Diff

Source Snapshot 필요분

Guide Original File

Converted PDF

Docling JSON

Markdown

Backup Artifact
```

---

# 28. Artifact Storage 후보

폐쇄망에서:

```text
NFS

NAS

S3-compatible Internal Object Storage

Shared File System
```

등을 사용할 수 있다.

---

# 29. Artifact URI

Application에서 물리 경로 대신 URI를 사용한다.

```text
artifact://guide/DOC-100/original

artifact://build/EXEC-200/log
```

---

# 30. Artifact HA

Artifact Storage 자체도 중앙 공유 방식으로 운영하여 Server Node 장애와 무관하게 접근 가능해야 한다.

---

# 31. Local Artifact

Local Agent 전용:

```text
Temporary Diff

Backup

Build Full Log

Test Full Log
```

은 로컬에 먼저 저장할 수 있다.

필요한 Artifact만 중앙으로 전송한다.

---

# 32. Inference Architecture

Agent Runtime과 LLM Serving은 분리한다.

```text
Agent Runtime
     ↓
Inference Gateway
     ↓
Inference Server Pool
```

---

# 33. Inference Gateway 책임

```text
Model Routing

Concurrency Control

Queue

Timeout

Retry

Health Check

Model Profile Mapping

Metrics
```

---

# 34. Inference Server

폐쇄망 내부에서:

```text
vLLM

또는 승인된 Internal Inference Engine
```

사용 가능.

Agent가 직접 inference endpoint를 관리하지 않는다.

---

# 35. Model Pool

예:

```text
Reasoning Model Pool

Coding Model Pool

Small Utility Model Pool
```

로 분리 가능.

---

# 36. Model Routing

예:

```text
Senior Developer
→ reasoning-high

Implementation
→ coding

Guide Query Analyzer
→ utility
```

모델 이름은 Agent Definition과 직접 결합하지 않는다.

---

# 37. GPU 자원 부족

Inference Gateway는 요청 Queue를 관리한다.

상태:

```text
QUEUED

RUNNING

TIMEOUT

REJECTED_RESOURCE_LIMIT
```

---

# 38. GPU 서버 HA

필요한 경우 같은 Model을 여러 Worker에 배포한다.

```text
Inference Gateway
 ├─ Worker 1
 ├─ Worker 2
 └─ Worker 3
```

---

# 39. Model Loading

대형 Model은 요청마다 Load하지 않는다.

상시 Serving 또는 운영 정책에 따른 Preload 사용.

---

# 40. Guide Ingestion Deployment

Guide Ingestion은 Runtime Agent 서버와 분리하는 것을 권장한다.

```text
Guide Admin
    ↓
Ingestion API
    ↓
Job Queue
    ↓
Conversion Workers
    ├─ LibreOffice
    ├─ Docling
    └─ LLM Enrichment
```

---

# 41. Ingestion 분리 이유

```text
문서 변환 CPU/Memory 사용 큼

OCR 가능

LibreOffice Hang 가능

Runtime 사용자 요청과 SLA 다름
```

---

# 42. Conversion Worker

Container로 격리한다.

```text
LibreOffice Worker

Docling Worker
```

---

# 43. Worker Scale-out

Queue depth를 기준으로 Worker 수를 늘릴 수 있다.

단 GPU가 필요한 Docling/OCR 구성은 자원 정책을 별도 적용한다.

---

# 44. Guide Runtime Search

Runtime Guide Search는 읽기 부하가 많고 Ingestion과 성격이 다르다.

필요하면 별도 Service로 분리한다.

---

# 45. Nexus

Nexus는 기존 내부 인프라를 사용한다.

onCode 배포에 Nexus 자체를 포함시키지 않는다.

onCode는 Adapter를 통해 연결한다.

---

# 46. Vulnerability DB

기존 구축 또는 별도 내부 DB를 사용한다.

Runtime 서비스와 데이터 반입 Process를 분리한다.

---

# 47. Security Data Import

```text
외부 보안망
   ↓ 보안 반입 절차
폐쇄망
   ↓
Vulnerability Data Importer
   ↓
Internal Vulnerability DB
```

onCode Runtime이 외부 인터넷에 연결하지 않는다.

---

# 48. Authentication Infrastructure

기존:

```text
LDAP

AD

Internal IdP
```

를 재사용한다.

IdP 장애 시 기존 Session과 신규 Login 정책을 구분한다.

---

# 49. IdP 장애

신규 Login:

```text
BLOCK
```

기존 Session:

```text
정책에 따라 일정 기간 유지
```

가능.

---

# 50. Local Agent 설치

지원 OS:

```text
Windows

macOS

Linux
```

실제 사용자 환경에 따라 순차 지원 가능.

---

# 51. Local Agent 패키징

예:

```text
Windows Installer

macOS Package

Linux Package / Binary
```

폐쇄망 내부 배포 시스템 사용.

---

# 52. Local Agent Auto-update

Public Internet Auto-update는 사용하지 않는다.

```text
Internal Update Server

Package Repository

Software Distribution System
```

등을 사용한다.

---

# 53. Version 정책

Server는 다음 Version을 관리한다.

```text
Minimum Supported Agent Version

Recommended Agent Version

Minimum IDE Extension Version

Protocol Version
```

---

# 54. 구버전 차단

Security 문제가 있는 Agent는:

```text
BLOCKED
```

상태로 접속 금지 가능.

---

# 55. IDE Extension 배포

폐쇄망 내부 패키지로 제공.

```text
VSIX

IntelliJ Plugin

Eclipse Update Site / Plugin
```

---

# 56. Server 배포

권장:

```text
Container Image
```

기반.

폐쇄망 내부 Container Registry에서 배포한다.

---

# 57. Container Registry

모든 Server Image는 내부 Registry를 사용한다.

외부 Docker Hub 등 Runtime 접근 금지.

---

# 58. Image 구성

예:

```text
oncode-server

guide-ingestion-api

guide-docling-worker

guide-libreoffice-worker

inference-gateway
```

---

# 59. Container Orchestration

조직 표준에 따라:

```text
Kubernetes

OpenShift

Docker Compose

VM + systemd
```

사용 가능.

---

# 60. MVP 배포

초기 개발/MVP에서는:

```text
Docker Compose
또는
단일 VM + Container
```

로도 충분하다.

---

# 61. Production

사용자 수와 운영 요구가 높아지면 Kubernetes 계열이 적합할 수 있다.

하지만 Kubernetes 자체를 필수 조건으로 두지 않는다.

---

# 62. Environment 분리

최소:

```text
DEV

TEST

PROD
```

환경 분리 권장.

---

# 63. 폐쇄망 개발 환경

DEV에서도 가능한 한 Production과 동일한 내부 의존성을 사용한다.

```text
Nexus

Internal Git

Internal LLM
```

---

# 64. Configuration

환경 설정은 Application Image에서 분리한다.

예:

```text
DB Connection

Inference Endpoint

Nexus Endpoint

Guide Config

Policy
```

---

# 65. Secret 관리

Secret을 일반 Config File/Image에 포함하지 않는다.

조직 표준 Secret Store 또는 OS/Kubernetes Secret을 사용한다.

---

# 66. Secret 종류

```text
DB Credential

Nexus Credential

IdP Secret

Internal Service Token

Certificate Key
```

---

# 67. Environment Variable

Secret을 Environment Variable로 전달할 수도 있으나 Process Dump/Log 노출 가능성을 고려한다.

조직 보안 정책에 따른다.

---

# 68. TLS

다음 구간 TLS 권장:

```text
IDE/Local Agent ↔ Server

Server ↔ Internal Services

Server ↔ PostgreSQL

Server ↔ Inference Gateway
```

---

# 69. 내부 CA

폐쇄망 내부 CA를 사용하여 인증서를 배포할 수 있다.

---

# 70. mTLS

고보안 환경에서는:

```text
Local Agent ↔ Server

Service ↔ Service
```

mTLS를 단계적으로 적용한다.

---

# 71. Network Zone

가능하면 논리 Zone을 분리한다.

```text
Developer Network

Application Network

Inference Network

Data Network

Management Network
```

---

# 72. Network ACL

예:

```text
Developer PC
→ onCode Gateway만 접근

onCode Server
→ DB / Nexus / IdP / LLM 접근

Developer PC
→ Inference Server 직접 접근 금지
```

---

# 73. Local Agent Inbound

외부 Network에서 Local Agent로 inbound 연결하는 구조는 사용하지 않는다.

Local Agent가 outbound connection을 생성한다.

---

# 74. Firewall

필요한 내부 Port만 허용한다.

---

# 75. Health Check

Server:

```text
/liveness

/readiness
```

지원.

---

# 76. Liveness

프로세스 자체 정상 여부.

---

# 77. Readiness

필수 의존성 기준으로 실제 요청 처리 가능 여부 판단.

예:

```text
DB unavailable
→ not ready
```

---

# 78. Component Health

하위 서비스 상태:

```text
HEALTHY

DEGRADED

UNAVAILABLE
```

---

# 79. Dependency별 영향

예:

```text
Nexus DOWN
→ 신규 Dependency 작업만 제한

Guide Search DOWN
→ 정책에 따라 구현 Workflow 제한

LLM DOWN
→ Agent Workflow 불가

Audit DOWN
→ High-risk operation 제한
```

---

# 80. Graceful Degradation

전체 서비스를 무조건 종료하는 대신 기능별 DEGRADED Mode를 지원한다.

---

# 81. Circuit Breaker

다음 외부 내부 서비스에 적용 가능.

```text
Inference

Nexus

IdP

Guide Service
```

---

# 82. Retry 정책

Network Timeout 등 일시적 오류:

```text
limited retry
```

정책.

무한 재시도 금지.

---

# 83. Retry Storm 방지

장애 상황에서 모든 Server Instance가 동시에 반복 요청하지 않도록:

```text
Exponential Backoff

Jitter

Circuit Breaker
```

사용.

---

# 84. Queue

비동기 작업:

```text
Guide Ingestion

Audit Event Delivery

Workflow Background Task
```

에 Durable Queue를 사용할 수 있다.

---

# 85. Runtime Workflow Queue

LLM/Tool 자원을 초과한 요청은 Queue에 둔다.

---

# 86. Queue 우선순위

필요하면:

```text
INTERACTIVE

BACKGROUND

ADMIN
```

로 구분.

개발자 Interactive 요청을 우선한다.

---

# 87. Resource Quota

사용자 또는 프로젝트별:

```text
Concurrent Workflow

Concurrent Agent Execution

LLM Request

Guide Ingestion
```

제한 가능.

---

# 88. Local Build Resource

Build/Test는 Local PC에서 실행되므로 중앙 서버 CPU를 소비하지 않는다.

이는 onCode 구조의 주요 장점이다.

---

# 89. 중앙 서버 주요 부하

```text
LLM orchestration

Context

Search

Workflow

Persistent connections
```

이다.

---

# 90. Capacity Planning

주요 변수:

```text
Concurrent users

Local Agent connections

Concurrent Workflows

LLM concurrency

Average source transfer

Guide search QPS
```

---

# 91. Persistent Connection Capacity

Local Agent 연결 수가 많아지면 Gateway/Server의 Connection Capacity를 별도로 산정한다.

---

# 92. Scale-out 기준

Server:

```text
CPU

Memory

Active Connections

Workflow Queue
```

Inference:

```text
GPU utilization

VRAM

Queue depth
```

Worker:

```text
Job Queue
```

를 기준으로 한다.

---

# 93. Horizontal Scale

onCode Server는 Stateless 원칙 덕분에 Scale-out 가능해야 한다.

---

# 94. Vertical Scale

Inference 서버는 GPU 특성상 Vertical Scaling이 더 중요할 수 있다.

---

# 95. Session Store

Active Session/Connection Registry는:

```text
RDBMS

Redis-compatible Store
```

등을 사용할 수 있다.

MVP는 PostgreSQL로 시작 가능.

---

# 96. Redis 필요 시점

다음이 커질 때 검토:

```text
수천 Persistent Connections

Fast Session Lookup

Distributed Locks

Pub/Sub
```

---

# 97. Message Broker 필요 시점

다음이 필요할 때 도입 가치가 높다.

```text
Cross-node Local Agent Dispatch

Durable Async Jobs

High Throughput Events
```

---

# 98. 초기 복잡성 억제

MVP부터:

```text
Kafka
Redis
Kubernetes
Multiple Microservices
```

를 모두 도입할 필요는 없다.

---

# 99. MVP 권장 구성

```text
1~2 onCode Server

PostgreSQL

Shared Artifact Storage

Inference Gateway

Internal LLM Server

Existing Nexus

Existing IdP

Guide Ingestion Worker
```

정도로 시작 가능.

---

# 100. Production 권장 구조

```text
Load Balancer

2+ onCode Server

HA PostgreSQL

Shared Artifact Store

Inference Gateway + GPU Pool

Guide Ingestion Worker Pool

Central Observability
```

---

# 101. Backup 대상

중요도 순:

```text
PostgreSQL

Guide Source / Release

Policy

Audit

Artifact
```

---

# 102. 반드시 백업할 데이터

```text
Workflow State

Context

Guide Repository

User/Project Metadata

Audit

Policy
```

---

# 103. 재생성 가능한 데이터

예:

```text
Project Index
```

는 Local Workspace에서 재생성 가능하다.

다만 중앙 Catalog도 운영 편의를 위해 백업 가능.

---

# 104. Build/Test Log

보존 정책에 따라 백업 선택.

---

# 105. RPO / RTO

프로젝트 요구사항에 따라 정의해야 한다.

예시 개념:

```text
RPO
→ 허용 가능한 데이터 유실 시점

RTO
→ 서비스 복구 목표 시간
```

실제 숫자는 운영 SLA에서 확정한다.

---

# 106. Disaster Recovery 우선순위

```text
1. DB
2. Authentication 연결
3. onCode Runtime
4. Inference
5. Artifact
6. Guide Ingestion
7. Observability
```

Guide Ingestion은 Runtime보다 복구 우선순위가 낮을 수 있다.

---

# 107. Server 전체 장애

Local Agent는:

```text
DISCONNECTED
```

상태.

High Risk/Agent Workflow는 중지.

Local Workspace는 그대로 유지된다.

---

# 108. 복구 후

```text
Reconnect
↓
Session validation
↓
Workspace revision check
↓
Pending workflow
↓
Resume
```

---

# 109. Workflow Recovery

상태:

```text
WAITING_USER_INPUT
WAITING_APPROVAL
WAITING_LOCAL_AGENT
```

은 그대로 복원할 수 있어야 한다.

---

# 110. 실행 중 Agent 장애

Agent Runtime Process 장애 시 LLM 호출 자체를 복구하지 않는다.

Task 상태를 확인 후 재실행한다.

---

# 111. 실행 중 Tool 장애

Tool Call이 실제 실행됐는지 불명확할 수 있다.

따라서 상태 확인이 중요하다.

예:

```text
git.commit
```

Timeout 후 Repository 상태 확인.

---

# 112. Unknown Outcome

상태:

```text
EXECUTION_OUTCOME_UNKNOWN
```

을 정의할 수 있다.

자동 재실행하지 않고 상태를 조회한다.

---

# 113. Apply Recovery

`workspace.apply_changes` 중 연결이 끊기면:

```text
change_set_id
idempotency_key
```

로 Local Agent 상태를 확인한다.

---

# 114. Local Agent 상태 조회

재접속 시:

```text
execution status

change set status
```

를 동기화한다.

---

# 115. Upgrade 전략

서버 Upgrade는 Rolling 방식 권장.

```text
Server A drain
↓
upgrade
↓
Server A ready
↓
Server B
```

---

# 116. Connection Draining

Persistent Connection이 있는 서버 Upgrade 전:

```text
new connections 차단

existing Local Agent reconnect 유도
```

방식.

---

# 117. Protocol Compatibility

Rolling Upgrade 동안 구버전/신버전 서버가 동시에 존재할 수 있으므로 Minor Version 호환성을 유지한다.

---

# 118. Database Migration

DB Schema Migration은 backward-compatible 방식 권장.

순서:

```text
Expand

Deploy

Migrate

Contract
```

---

# 119. Zero-downtime Migration

가능하면 새 Column 추가 → Application 전환 → 구 Column 제거 순서.

---

# 120. Local Agent Upgrade

Server와 Protocol 호환 가능한 범위에서 점진적 Upgrade를 허용한다.

---

# 121. Forced Upgrade

Critical 보안 문제 발생 시:

```text
minimum_agent_version
```

을 올려 구버전 차단.

---

# 122. Guide Release 배포

Guide Release는 Application Deployment와 별개로 배포한다.

```text
Guide Build
↓
Validation
↓
Activate
```

---

# 123. Guide Rollback

Application Rollback 없이 Guide Release만 이전 버전으로 되돌릴 수 있다.

---

# 124. Model Deployment

Model Version 변경도 Agent Application 배포와 분리한다.

Inference Gateway Model Profile을 통해 Routing을 변경한다.

---

# 125. Model Rollback

새 Model 품질 문제 발생 시 Profile을 이전 Model로 변경.

---

# 126. Prompt Deployment

Prompt Version도 Agent Definition과 함께 Version 관리.

Canary 또는 일부 테스트 Workflow로 검증 가능.

---

# 127. Feature Flag

새 기능을 단계적으로 활성화할 수 있다.

예:

```text
security_agent_enabled

partial_approval_enabled

new_retrieval_ranker
```

---

# 128. Feature Flag 범위

```text
Organization

Project

User group
```

단위 가능.

---

# 129. Canary

새 Agent/Prompt/Model을 일부 내부 사용자 대상으로 먼저 적용 가능.

---

# 130. 운영 모니터링

SPEC-13 기반 Dashboard 사용.

주요:

```text
Server Health

Workflow Queue

Local Agent Connectivity

LLM Queue

Database

Guide Search

Nexus

Security DB Freshness
```

---

# 131. Alert

중요 Alert:

```text
DB unavailable

Audit unavailable

LLM pool unavailable

Nexus unavailable

Vulnerability DB stale

Local Agent disconnect spike

Disk nearly full
```

---

# 132. Disk 관리

특히:

```text
Artifact

Local Log

Guide Original

Build/Test Log
```

Storage Capacity 모니터링 필요.

---

# 133. Artifact Cleanup

Retention Policy 기반 정리.

---

# 134. PostgreSQL Maintenance

운영:

```text
Vacuum

Analyze

Index maintenance

Partition maintenance
```

필요.

---

# 135. Audit Partition

장기 보존 시 날짜 Partition 권장.

---

# 136. Guide Search Index

Guide Release 활성화 전 Search Index 생성과 Validation을 완료한다.

---

# 137. Project Catalog 유지

Local Agent가 장시간 연결되지 않은 Workspace는 중앙 Catalog가 오래될 수 있다.

상태:

```text
ACTIVE

STALE

OFFLINE
```

관리.

---

# 138. Stale Project Catalog

검색 가능하더라도 실제 구현 전에 Local Workspace Revision을 재검증한다.

---

# 139. Multi-site

초기에는 단일 폐쇄망 Site를 전제로 한다.

여러 물리 Site가 필요해지면:

```text
Regional Server

Central Control

Guide/Policy Replication
```

등을 후속 설계한다.

---

# 140. Air-gap Update Package

외부에서 반입되는 항목은 패키지화한다.

예:

```text
Application Container Images

Model Files

Vulnerability Data

License Data

Python/npm/Maven Packages

IDE Extension

Local Agent Installer
```

---

# 141. 반입 검증

반입 Package는:

```text
Hash

Signature

Version

Manifest
```

검증 후 내부 Registry/Nexus에 등록한다.

---

# 142. Software Bill of Materials

onCode 자체 배포 Artifact에도 SBOM을 생성하는 것을 권장한다.

---

# 143. Container Security

Server Image:

```text
Non-root

Minimal Base Image

Read-only filesystem where possible

No unnecessary tools
```

권장.

---

# 144. LibreOffice Container

문서 변환은 untrusted document를 처리하므로 특히 격리한다.

```text
No external network

CPU/Memory limit

Timeout

Temporary filesystem
```

---

# 145. Docling Container

유사하게 Resource Limit과 Network Restriction 적용.

---

# 146. Runtime Server 권한

Runtime Server가 개발자 Workspace에 직접 접근할 권한은 없다.

구조적으로 Local Agent Tool을 통해서만 접근.

---

# 147. DB 계정 분리

서비스별 DB 계정을 나눌 수 있다.

예:

```text
runtime_user

guide_ingestion_user

audit_writer
```

---

# 148. 최소 권한

Runtime Service가 Audit Table Update/Delete 권한을 갖지 않도록 한다.

---

# 149. Security Patch

폐쇄망에서도 Application/Library 보안 Patch 운영 절차가 필요하다.

```text
External analysis

Approved import

Internal Nexus/Registry

Test

Production deploy
```

---

# 150. Dependency Pinning

onCode Server Build도 모든 Dependency를 내부 Nexus에서 재현 가능하게 해야 한다.

---

# 151. Build Reproducibility

Server/Local Agent 패키지 빌드는:

```text
Source revision

Dependency versions

Container base image

Build metadata
```

로 재현 가능해야 한다.

---

# 152. Release Manifest

각 onCode Release에:

```text
Server version

Local Agent version

Protocol version

DB schema version

Prompt versions

Guide compatibility

Supported extension versions
```

등을 기록한다.

---

# 153. 운영 Runbook

다음 Runbook을 작성하는 것을 권장한다.

```text
Server restart

DB failover

Inference failure

Nexus failure

Guide rollback

Agent upgrade

Session problem

Audit failure
```

---

# 154. Incident 운영

장애 발생:

```text
Detect

Alert

Incident ID

Mitigate

Recover

Postmortem
```

SPEC-13 Incident와 연결.

---

# 155. Maintenance Mode

DB migration 등 운영 작업 중:

```text
GENERAL_QA only

New workflow blocked

High-risk tool blocked
```

등 제한 가능.

---

# 156. Graceful Shutdown

Server 종료 전:

```text
stop new workflow assignment

finish or checkpoint current transition

release workflow lease

drain connections
```

수행.

---

# 157. Local Agent Graceful Shutdown

```text
running tool 확인

server disconnect event

local state flush
```

수행.

---

# 158. Time Sync

Server, DB, Local Agent, Inference Node 모두 내부 NTP로 시간 동기화 권장.

Audit/Approval/Trace 일관성 때문.

---

# 159. Config 관리

환경별 Config를 Git 또는 내부 Config Repository로 Version 관리할 수 있다.

Secret은 별도 관리.

---

# 160. Policy 배포

Organization Policy 변경:

```text
Draft

Validate

Version

Publish

Local Agent Sync
```

절차 권장.

---

# 161. Policy Rollback

정책 오류 시 이전 Version 복원 가능.

---

# 162. Policy Compatibility

구버전 Local Agent가 새 Policy Type을 해석하지 못하면 High-risk Tool 차단.

---

# 163. Capacity 테스트

Production 전 다음 부하 테스트가 필요하다.

```text
Concurrent users

Persistent Local Agent connections

Agent request queue

Guide Search

DB Context writes

Workflow transitions
```

---

# 164. LLM 부하 테스트

특히:

```text
Concurrent prompts

Long context

Coding generation

Review parallelization
```

테스트.

---

# 165. Chaos / Failure Test

다음 장애를 의도적으로 테스트할 가치가 있다.

```text
Server node kill

DB failover

Inference restart

Local Agent disconnect

Nexus timeout

Artifact store unavailable
```

---

# 166. Resume 테스트

onCode의 핵심 신뢰성 검증 항목이다.

```text
WAITING_APPROVAL에서 Server 재시작

BUILDING 중 연결 단절

Local Agent reconnect

Workspace modified during disconnect
```

테스트.

---

# 167. Backup Restore Drill

Backup이 있다는 것보다 실제 Restore가 되는지 정기 검증해야 한다.

---

# 168. 운영 데이터 보존

환경별로 Retention 정책을 정의한다.

```text
Audit

Logs

Artifacts

Guide Versions

Workflow History
```

---

# 169. Development 환경 데이터

DEV에서는 보존 기간을 짧게 설정 가능.

---

# 170. Production 데이터

승인/감사 이력은 조직 정책에 맞춰 장기 보존.

---

# 171. Recommended Deployment Profiles

## Profile A — Developer/MVP

```text
1 onCode Server

1 PostgreSQL

1 Inference Server

Existing Nexus

Local Artifact Directory/NFS
```

---

## Profile B — Team Pilot

```text
2 onCode Server

PostgreSQL Primary/Standby

Shared Artifact Store

Inference Gateway

1~N GPU Workers

Guide Ingestion Workers
```

---

## Profile C — Production

```text
Load Balancer

3+ Runtime Server

HA PostgreSQL

Distributed Session/Connection Registry

Message Broker

Shared Artifact Storage

Inference Gateway + GPU Pool

Guide Worker Pool

Central Monitoring
```

---

# 172. MVP에서 하지 않을 것

초기부터 필수로 하지 않는 것:

```text
Full Microservice decomposition

Multi-region

Complex Service Mesh

Kafka-scale Event Architecture

Automatic GPU autoscaling

Active-active multi-datacenter
```

---

# 173. 확장 방향

사용자/부하가 증가하면 순차 도입:

```text
Server Scale-out

Redis / Connection Registry

Message Broker

Dedicated Guide Search

Dedicated Agent Workers

Inference Pool 확장
```

---

# 174. 배포 구성 예

```text
Infrastructure

├─ gateway/
│  └─ load-balancer
│
├─ runtime/
│  ├─ oncode-server-1
│  └─ oncode-server-2
│
├─ data/
│  ├─ postgres
│  └─ artifact-storage
│
├─ inference/
│  ├─ inference-gateway
│  ├─ gpu-worker-1
│  └─ gpu-worker-2
│
├─ guide/
│  ├─ ingestion-api
│  ├─ docling-worker
│  └─ libreoffice-worker
│
└─ observability/
   ├─ metrics
   ├─ logs
   └─ dashboard
```

---

# 175. Runtime Request 흐름

```text
Developer
   ↓
IDE
   ↓
Local Agent
   ↓
Gateway
   ↓
Runtime Server
   ↓
Workflow / Agent
   ↓
Inference / DB / Guide
   ↓
Tool Gateway
   ↓
Local Agent
```

---

# 176. Server 장애 흐름

```text
Server A failure
   ↓
Load Balancer removes A
   ↓
Local Agents reconnect
   ↓
Connection Registry update
   ↓
Workflow lease expires
   ↓
Server B resumes
```

---

# 177. DB 장애 흐름

```text
DB primary failure
   ↓
DB failover
   ↓
Runtime not-ready temporarily
   ↓
DB reconnect
   ↓
Workflow state reload
   ↓
Resume
```

---

# 178. Inference 장애 흐름

```text
Model Worker failure
   ↓
Inference Gateway health detect
   ↓
Route to another worker
```

대체 Worker가 없으면 Workflow를 Retry/Wait 상태로 전환.

---

# 179. Nexus 장애 흐름

```text
Nexus unavailable
   ↓
Dependency Service degraded
   ↓
Existing-dependency work can continue
   ↓
New dependency resolution paused
```

---

# 180. Guide Ingestion 장애

Runtime Published Guide에는 영향이 없어야 한다.

```text
Ingestion DOWN
≠
Runtime Guide Search DOWN
```

---

# 181. Release 배포 순서

권장:

```text
1. DB compatible migration
2. Runtime Server deploy
3. Worker deploy
4. Inference config
5. Local Agent compatibility enable
6. IDE Extension rollout
```

---

# 182. Rollback 순서

```text
Application rollback

Config rollback

Prompt/model routing rollback

Guide release rollback
```

각 요소를 독립적으로 되돌릴 수 있도록 한다.

---

# 183. Deployment Audit

다음 운영 행위도 Audit한다.

```text
Server deployment

Policy publish

Guide release activate

Model profile change

Agent minimum version change
```

---

# 184. 운영 권한

Deployment와 Policy/Admin 권한을 일반 Developer와 분리한다.

---

# 185. Production Secure Defaults

```text
TLS enabled

External internet blocked

Nexus only

Audit enabled

High-risk fail-closed

2+ runtime instances

DB backup enabled

Guide auto-publish disabled
```

---

# 186. MVP Secure Defaults

```text
Single runtime acceptable

PostgreSQL backup

TLS

Internal Nexus

Code/Git approvals

Audit

No external internet
```

---

# 187. 운영 준비 체크리스트

Production 전 확인:

```text
Authentication

Project RBAC

TLS

Nexus connection

Vulnerability DB freshness

Inference health

Backup

Restore

Audit

Local Agent reconnect

Workflow resume

Guide rollback

DB failover
```

---

# 188. 핵심 설계 결정

onCode Deployment / HA / Operations Architecture v1의 핵심 결정은 다음과 같다.

1. Runtime Server는 가능한 한 Stateless하게 설계한다.
2. Workflow/Context/Pending Approval을 프로세스 메모리에만 보관하지 않는다.
3. 초기에는 논리적 모듈화를 유지한 Modular Monolith 형태를 권장한다.
4. Agent마다 별도 Microservice를 만들지 않는다.
5. Runtime Server는 최소 2개 이상의 HA 구성을 지원하도록 설계한다.
6. Local Agent Persistent Connection과 논리 Session을 분리한다.
7. Connection Registry를 통해 Local Agent가 연결된 Server Node를 추적한다.
8. Cross-node Tool Dispatch가 가능하도록 내부 Messaging 경로를 고려한다.
9. Workflow Lease/Optimistic Locking으로 Server 장애 후 Resume가 가능해야 한다.
10. 파일 적용/Git 등 부작용이 있는 Tool에는 Idempotency를 적용한다.
11. PostgreSQL을 핵심 상태 저장소로 사용하고 Production에서는 HA 구성을 고려한다.
12. 대용량 Artifact는 DB와 분리된 Shared Artifact Storage를 사용한다.
13. Agent Runtime과 LLM Inference Runtime을 분리한다.
14. Inference Gateway가 모델 Routing, Queue, Timeout, Metrics를 담당한다.
15. Guide Ingestion은 Runtime 서버와 별도 Worker 계층으로 분리한다.
16. Nexus/IdP/Git 등 기존 내부 인프라는 재구현하지 않고 연동한다.
17. 폐쇄망 Runtime에서 외부 인터넷 접근을 전제로 하지 않는다.
18. Local Agent/IDE/Server 업데이트는 내부 배포 체계를 사용한다.
19. 서버와 Local Agent 간 연결은 outbound persistent TLS 방식을 유지한다.
20. 서버 장애 시 Local Agent가 재접속하고 Workflow를 중앙 상태에서 Resume한다.
21. 기능별 장애를 DEGRADED Mode로 격리한다.
22. Guide Ingestion 장애가 Runtime Guide Search에 영향을 주지 않게 한다.
23. 신규 Dependency 기능은 Nexus 장애 시 중단하되 기존 Dependency 기반 작업은 가능한 범위에서 계속할 수 있다.
24. Upgrade는 Rolling Deployment와 backward-compatible DB migration을 지향한다.
25. Server, Local Agent, Protocol, Prompt, Model, Guide Release를 독립 Versioning한다.
26. Guide/Policy/Model Routing을 Application 배포와 독립적으로 Rollback할 수 있게 한다.
27. 운영 복잡도를 줄이기 위해 MVP부터 Kubernetes/Kafka 등 모든 분산 인프라를 도입하지 않는다.
28. 부하가 증가하면 Connection Registry → Message Broker → Dedicated Worker 순으로 확장한다.
29. Backup뿐 아니라 Restore와 Workflow Resume Drill을 정기 검증한다.
30. Production 운영 전 Local Agent 재접속, DB Failover, Server Kill, Inference 장애 시나리오를 검증한다.